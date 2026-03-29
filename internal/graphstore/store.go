/**
 * @file store.go
 * @brief Neo4j driver lifecycle and session helpers for read/write Cypher execution.
 * @auther rajeshkurup@live.com
 */
package graphstore

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

/**
 * @brief Wraps a Neo4j driver and logical database name for all graph operations.
 */
type Store struct {
	driver   neo4j.DriverWithContext
	database string
}

/**
 * @brief Constructs a Store: creates driver, verifies connectivity, returns usable handle or error.
 * @param ctx context for driver verification.
 * @param uri Neo4j Bolt URI (e.g. bolt://localhost:7687).
 * @param user Neo4j username.
 * @param password Neo4j password.
 * @param database logical database name (e.g. neo4j).
 * @return *Store on success; error if driver creation or connectivity check fails.
 */
func New(ctx context.Context, uri, user, password, database string) (*Store, error) {
	auth := neo4j.BasicAuth(user, password, "")
	drv, err := neo4j.NewDriverWithContext(uri, auth)
	if err != nil {
		return nil, err
	}
	if err := drv.VerifyConnectivity(ctx); err != nil {
		_ = drv.Close(ctx)
		return nil, fmt.Errorf("neo4j connectivity: %w", err)
	}
	return &Store{driver: drv, database: database}, nil
}

/**
 * @brief Closes the underlying Neo4j driver; safe on nil receiver or nil driver.
 * @param s the store (receiver).
 * @param ctx context for close operation.
 * @return Error from driver Close, or nil.
 */
func (s *Store) Close(ctx context.Context) error {
	if s == nil || s.driver == nil {
		return nil
	}
	return s.driver.Close(ctx)
}

/**
 * @brief Opens a new Neo4j session for the store database in read or write mode.
 * @param s the store (receiver).
 * @param ctx context passed to driver.NewSession.
 * @param write if true, uses write access mode; otherwise read.
 * @return A SessionWithContext that the caller must close (writeResult/readResult close it).
 */
func (s *Store) session(ctx context.Context, write bool) neo4j.SessionWithContext {
	mode := neo4j.AccessModeRead
	if write {
		mode = neo4j.AccessModeWrite
	}
	return s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.database,
		AccessMode:   mode,
	})
}

/**
 * @brief Runs a write transaction: executes cypher with params and collects all records.
 * @param ctx context for session and transaction.
 * @param sess Neo4j session (closed by this function via defer).
 * @param cypher Cypher statement.
 * @param params bound parameters map.
 * @return Collected records, or error from run/collect.
 */
func writeResult(ctx context.Context, sess neo4j.SessionWithContext, cypher string, params map[string]any) ([]*neo4j.Record, error) {
	defer sess.Close(ctx)
	res, err := sess.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		r, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		return r.Collect(ctx)
	})
	if err != nil {
		return nil, err
	}
	recs, _ := res.([]*neo4j.Record)
	return recs, nil
}

/**
 * @brief Runs a read transaction: executes cypher with params and collects all records.
 * @param ctx context for session and transaction.
 * @param sess Neo4j session (closed by this function via defer).
 * @param cypher Cypher query.
 * @param params bound parameters map.
 * @return Collected records, or error from run/collect.
 */
func readResult(ctx context.Context, sess neo4j.SessionWithContext, cypher string, params map[string]any) ([]*neo4j.Record, error) {
	defer sess.Close(ctx)
	res, err := sess.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		r, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		return r.Collect(ctx)
	})
	if err != nil {
		return nil, err
	}
	recs, _ := res.([]*neo4j.Record)
	return recs, nil
}
