# Neo4j Topology Schema Documentation

## 1️⃣ Node Labels and Properties

| Label | Purpose | Key Properties |
|-------|---------|----------------|
| **Application** | Represents apps/services in your topology | `id` (unique), `name`, `tier` (frontend/backend), `owner`, `criticality` |
| **Storage** | Storage resources (DBs, disks) | `id` (unique), `name`, `type` (SQL/NoSQL), `capacity` |
| **Network** | Network elements (switches, routers, VPCs) | `id` (unique), `name`, `type`, `location` |
| **IncidentTicket** | Operational incidents | `id` (unique), `severity`, `status`, `startTime`, `endTime` |
| **ChangeTicket** | Changes impacting topology | `id` (unique), `description`, `startTime`, `endTime`, `status` |
| **RCATicket** | Root cause analysis | `id` (unique), `description`, `status` |
| **Action** | Corrective action linked to RCA | `id` (unique), `description`, `owner`, `status` |
| **Anomaly** | Detected abnormal behavior | `id` (unique), `type`, `severity`, `status`, `startTime`, `endTime` |
| **Call** *(optional)* | Reified app-to-app interactions | `id` (unique), `latency`, `errorRate` |

## 2️⃣ Relationships and Properties

| Relationship | Direction | Properties | Notes |
|--------------|----------|-----------|-------|
| `(:Application)-[:CALLS]->(:Application)` | app → app | `startTime`, `endTime` | For dependency / traversal |
| `(:Application)-[:USES_STORAGE]->(:Storage)` | app → storage | `startTime`, `endTime` | Layered dependency |
| `(:Application)-[:CONNECTS_TO]->(:Network)` | app → network | optional | Connectivity info |
| `(:Storage)-[:STORED_ON_NETWORK]->(:Network)` | storage → network | optional | Layered dependency |
| `(:IncidentTicket)-[:IMPACTS]->(:Application|Storage|Network)` | ticket → node | `startTime`, `endTime`, `role` | Time-aware impact |
| `(:ChangeTicket)-[:AFFECTS]->(:Application|Storage|Network)` | ticket → node | `startTime`, `endTime`, `description` | Change analysis |
| `(:RCATicket)-[:ROOT_CAUSE_OF]->(:IncidentTicket)` | rca → incident | none | Links RCA to incidents |
| `(:RCATicket)-[:HAS_ACTION]->(:Action)` | rca → action | none | Corrective actions |
| `(:Anomaly)-[:DETECTED_ON]->(:Application|Storage|Network)` | anomaly → node | `startTime`, `endTime`, `severity` | Used for root cause |
| `(:Call)-[:TO]->(:Application)` | interaction → app | latency, errorRate | Optional for SRE analysis |
| `(:Application)-[:DEPENDS_ON_TRANSITIVE]->(:Network|Storage|Application)` | precomputed | none | Optional derived edge for blast radius optimization |

## 3️⃣ Constraints

```cypher
CREATE CONSTRAINT application_id IF NOT EXISTS FOR (a:Application) REQUIRE a.id IS UNIQUE;
CREATE CONSTRAINT storage_id IF NOT EXISTS FOR (s:Storage) REQUIRE s.id IS UNIQUE;
CREATE CONSTRAINT network_id IF NOT EXISTS FOR (n:Network) REQUIRE n.id IS UNIQUE;
CREATE CONSTRAINT incident_id IF NOT EXISTS FOR (i:IncidentTicket) REQUIRE i.id IS UNIQUE;
CREATE CONSTRAINT change_id IF NOT EXISTS FOR (c:ChangeTicket) REQUIRE c.id IS UNIQUE;
CREATE CONSTRAINT rca_id IF NOT EXISTS FOR (r:RCATicket) REQUIRE r.id IS UNIQUE;
CREATE CONSTRAINT anomaly_id IF NOT EXISTS FOR (a:Anomaly) REQUIRE a.id IS UNIQUE;
CREATE CONSTRAINT call_id IF NOT EXISTS FOR (c:Call) REQUIRE c.id IS UNIQUE;
```

## 4️⃣ Indexes

```cypher
CREATE INDEX app_name IF NOT EXISTS FOR (a:Application) ON (a.name);
CREATE INDEX storage_name IF NOT EXISTS FOR (s:Storage) ON (s.name);
CREATE INDEX network_name IF NOT EXISTS FOR (n:Network) ON (n.name);
CREATE INDEX incident_time IF NOT EXISTS FOR (i:IncidentTicket) ON (i.startTime, i.endTime);
CREATE INDEX anomaly_time IF NOT EXISTS FOR (a:Anomaly) ON (a.startTime, a.endTime);
CREATE INDEX incident_severity IF NOT EXISTS FOR (i:IncidentTicket) ON (i.severity);
CREATE INDEX anomaly_status IF NOT EXISTS FOR (a:Anomaly) ON (a.status);
```

## 5️⃣ Traversal Guidelines

- **Root Cause Analysis:** traverse CALLS / USES_STORAGE / CONNECTS_TO edges from impacted nodes, filter active anomalies, limit depth.
- **Blast Radius / Impact:** reverse-traverse edges from failed node, optionally use DEPENDS_ON_TRANSITIVE precomputed edges.
- **Ranking:** prioritize anomalies by severity and confidence.

## 6️⃣ Example Node JSON

**Application**
```json
{
  "id": "app-123",
  "name": "PaymentsService",
  "tier": "backend",
  "owner": "team-payments",
  "criticality": "high"
}
```

**IncidentTicket**
```json
{
  "id": "INC-101",
  "severity": "SEV1",
  "status": "resolved",
  "startTime": "2026-03-25T10:00:00",
  "endTime": "2026-03-25T11:00:00"
}
```

**Anomaly**
```json
{
  "id": "ANOM-1",
  "type": "latency_spike",
  "status": "active",
  "severity": "high",
  "startTime": "2026-03-25T09:55:00",
  "endTime": null
}
```

## 7️⃣ Optional Enhancements

- Derived edges for blast radius: `DEPENDS_ON_TRANSITIVE`
- Reified CALLS for latency/error metrics
- Label partitioning: `CriticalApplication` vs `NonCriticalApplication`
- Graph Data Science algorithms for advanced RCA

## 8️⃣ Summary

- Single database for both RCA and blast radius
- Lightweight topology edges for fast traversal
- Anomalies/tickets separate for time-aware analysis
- Derived edges precompute impact optionally
- Proper indexes & constraints ensure performance

