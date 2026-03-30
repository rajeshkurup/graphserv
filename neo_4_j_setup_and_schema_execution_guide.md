# Neo4j Setup and Schema Execution Guide

## 1️⃣ Connect to Neo4j

1. Start your Neo4j instance (Desktop or Aura).
2. Open Neo4j Browser or Neo4j Bloom.
3. Connect using:
   - URI: `bolt://localhost:7687`
   - Username: `neo4j`
   - Password: (your configured password)

---

## 2️⃣ Create Constraints

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

---

## 3️⃣ Create Indexes

```cypher
CREATE INDEX app_name IF NOT EXISTS FOR (a:Application) ON (a.name);
CREATE INDEX storage_name IF NOT EXISTS FOR (s:Storage) ON (s.name);
CREATE INDEX network_name IF NOT EXISTS FOR (n:Network) ON (n.name);

CREATE INDEX incident_time IF NOT EXISTS FOR (i:IncidentTicket) ON (i.startTime, i.endTime);
CREATE INDEX anomaly_time IF NOT EXISTS FOR (a:Anomaly) ON (a.startTime, a.endTime);

CREATE INDEX incident_severity IF NOT EXISTS FOR (i:IncidentTicket) ON (i.severity);
CREATE INDEX anomaly_status IF NOT EXISTS FOR (a:Anomaly) ON (a.status);
```

---

## 4️⃣ Create Sample Nodes

### Applications

```cypher
CREATE (a1:Application {id: 'app-123', name: 'PaymentsService', tier: 'backend', owner: 'team-payments', criticality: 'high'});
CREATE (a2:Application {id: 'app-124', name: 'OrdersService', tier: 'backend', owner: 'team-orders', criticality: 'medium'});
```

### Storage

```cypher
CREATE (s1:Storage {id: 'db-001', name: 'PaymentsDB', type: 'SQL', capacity: '500GB'});
CREATE (s2:Storage {id: 'db-002', name: 'OrdersDB', type: 'NoSQL', capacity: '1TB'});
```

### Network

```cypher
CREATE (n1:Network {id: 'net-01', name: 'VPC-East', type: 'VPC', location: 'us-east-1'});
CREATE (n2:Network {id: 'net-02', name: 'VPC-West', type: 'VPC', location: 'us-west-2'});
```

---

## 5️⃣ Create Relationships

### App to App

```cypher
MATCH (a1:Application {id:'app-123'}), (a2:Application {id:'app-124'})
CREATE (a1)-[:CALLS]->(a2);
```

### App to Storage

```cypher
MATCH (a:Application {id:'app-123'}), (s:Storage {id:'db-001'})
CREATE (a)-[:USES_STORAGE]->(s);
```

### App to Network

```cypher
MATCH (a:Application {id:'app-123'}), (n:Network {id:'net-01'})
CREATE (a)-[:CONNECTS_TO]->(n);
```

### Storage to Network

```cypher
MATCH (s:Storage {id:'db-001'}), (n:Network {id:'net-01'})
CREATE (s)-[:STORED_ON_NETWORK]->(n);
```

---

## 6️⃣ Create Anomalies and Tickets

### Anomaly

```cypher
MATCH (a:Application {id:'app-123'})
CREATE (anom:Anomaly {id:'ANOM-1', type:'latency_spike', severity:'high', status:'active', startTime: datetime()})
CREATE (anom)-[:DETECTED_ON]->(a);
```

### Incident

```cypher
MATCH (a:Application {id:'app-123'})
CREATE (inc:IncidentTicket {id:'INC-101', severity:'SEV1', status:'active', startTime: datetime()})
CREATE (inc)-[:IMPACTS]->(a);
```

### RCA and Action

```cypher
CREATE (rca:RCATicket {id:'RCA-1', description:'Investigate latency spike', status:'open'});
CREATE (action:Action {id:'ACT-1', description:'Restart service', owner:'team-payments', status:'pending'});
CREATE (rca)-[:HAS_ACTION]->(action);
MATCH (inc:IncidentTicket {id:'INC-101'}) CREATE (rca)-[:ROOT_CAUSE_OF]->(inc);
```

---

## 7️⃣ Optional: Precompute Blast Radius

```cypher
MATCH (a:Application)-[:USES_STORAGE]->(s:Storage)-[:STORED_ON_NETWORK]->(n:Network)
MERGE (a)-[:DEPENDS_ON_TRANSITIVE]->(s)
MERGE (a)-[:DEPENDS_ON_TRANSITIVE]->(n);
```

---

## 8️⃣ Testing Traversals

### Root Cause Query

```cypher
MATCH (start:Application {id: 'app-123'})
MATCH path = (start)-[:CALLS|USES_STORAGE|CONNECTS_TO|STORED_ON_NETWORK*1..5]->(n)
MATCH (a:Anomaly)-[:DETECTED_ON]->(n)
WHERE a.status = 'active'
RETURN n, a, path
LIMIT 50;
```

### Blast Radius Query

```cypher
MATCH (failed:Application {id:'app-123'})<-[:DEPENDS_ON_TRANSITIVE]-(impacted)
RETURN impacted;
```

---

## ✅ Summary

- Constraints ensure data integrity
- Indexes improve lookup and filtering performance
- Sample data helps validate schema
- Traversal queries support root cause and blast radius analysis

