# Building a Distributed Task Queue from Scratch

This diagram shows how to recreate the Asynq-like background job service without using the Asynq library, based on the studied architecture and features.

## Core Architecture for Scratch Implementation

```mermaid
graph TB
    subgraph "Client Applications"
        APP1[Server 1<br/>Go Application]
        APP2[Server 2<br/>Go Application]
        CRON_MGR[Cron Manager<br/>Custom Scheduler]
    end
    
    subgraph "Task Queue System - DIY Implementation"
        subgraph "Client Library"
            CLIENT[Task Client<br/>• Task Serialization<br/>• Redis Commands<br/>• Retry Logic]
        end
        
        subgraph "Redis Message Broker"
            REDIS[Redis Instance<br/>• Task Storage<br/>• Queue Management<br/>• Pub/Sub<br/>• Atomic Operations]
        end
        
        subgraph "Worker Server"
            MANAGER[Worker Manager<br/>• Pool Management<br/>• Health Monitoring]
            
            subgraph "Core Components to Build"
                POLLER[Task Poller<br/>• BRPOP from queues<br/>• Lease management<br/>• Heartbeat]
                SCHEDULER[Task Scheduler<br/>• Check due tasks<br/>• Move to pending<br/>• Timer-based]
                PROCESSOR[Task Processor<br/>• Execute handlers<br/>• Handle failures<br/>• Context management]
                MONITOR[Health Monitor<br/>• Redis connectivity<br/>• Worker status<br/>• Metrics collection]
            end
            
            subgraph "Worker Pool"
                W1[Worker Goroutine 1]
                W2[Worker Goroutine 2]
                WN[Worker Goroutine N]
            end
        end
        
        subgraph "Management Interface"
            API[REST API<br/>• Queue stats<br/>• Task management<br/>• Worker status]
            WEB[Web Dashboard<br/>• Queue monitoring<br/>• Task inspection]
        end
    end
    
    %% Connections
    APP1 --> CLIENT
    APP2 --> CLIENT
    CRON_MGR --> CLIENT
    
    CLIENT --> REDIS
    
    POLLER --> REDIS
    SCHEDULER --> REDIS
    PROCESSOR --> REDIS
    MONITOR --> REDIS
    
    MANAGER --> POLLER
    MANAGER --> SCHEDULER
    MANAGER --> PROCESSOR
    MANAGER --> MONITOR
    
    PROCESSOR --> W1
    PROCESSOR --> W2
    PROCESSOR --> WN
    
    API --> REDIS
    WEB --> API
```

## Implementation Components Breakdown

### 1. Client Library Implementation

```mermaid
graph LR
    subgraph "Client Library Functions"
        A[Task Definition<br/>• Type + Payload<br/>• Serialization<br/>• Validation] 
        B[Queue Operations<br/>• Enqueue immediate<br/>• Schedule future<br/>• Unique constraints]
        C[Redis Interface<br/>• Connection pool<br/>• Command execution<br/>• Error handling]
    end
    
    A --> B --> C
```

**Key Components to Build:**
- Task struct with JSON/protobuf serialization
- Redis client wrapper with connection pooling
- Enqueue methods: immediate, scheduled, unique
- Error handling and retry logic

### 2. Redis Data Structure Design

```mermaid
graph TB
    subgraph "Redis Keys Structure"
        subgraph "Task Queues"
            PQ[Pending Queue<br/>LIST: tasks:pending<br/>LPUSH new tasks<br/>BRPOP for workers]
            AQ[Active Queue<br/>LIST: tasks:active<br/>Track running tasks]
            SQ[Scheduled Queue<br/>ZSET: tasks:scheduled<br/>Score = execution time]
            RQ[Retry Queue<br/>ZSET: tasks:retry<br/>Score = retry time]
        end
        
        subgraph "Task Data"
            TD[Task Data<br/>HASH: task:id<br/>Store task payload<br/>metadata, status]
            UL[Unique Locks<br/>STRING: unique:hash<br/>SETEX for uniqueness<br/>TTL-based cleanup]
        end
        
        subgraph "Worker Management"
            WS[Worker Status<br/>HASH: worker:id<br/>Heartbeat, current task]
            LS[Lease System<br/>HASH: lease:queue<br/>Task ownership<br/>Expiry tracking]
        end
        
        subgraph "Statistics"
            ST[Stats Counters<br/>processed, failed<br/>Daily/total counts]
            PS[Pub/Sub Channels<br/>cancel:task:id<br/>pause:queue]
        end
    end
```

### 3. Worker Server Implementation

```mermaid
sequenceDiagram
    participant Main as Main Process
    participant Manager as Worker Manager
    participant Poller as Task Poller
    participant Scheduler as Task Scheduler
    participant Processor as Task Processor
    participant Redis as Redis
    
    Main->>Manager: Start worker server
    Manager->>Poller: Start task polling
    Manager->>Scheduler: Start scheduler loop
    Manager->>Processor: Initialize worker pool
    
    loop Every 5 seconds
        Scheduler->>Redis: ZPOPMIN scheduled/retry queues
        Scheduler->>Redis: LPUSH to pending if due
    end
    
    loop Continuous
        Poller->>Redis: BRPOP pending queue
        Redis-->>Poller: Return task ID
        Poller->>Redis: LPUSH to active queue
        Poller->>Redis: HSET lease with expiry
        Poller->>Processor: Submit task to worker pool
        
        Processor->>Processor: Execute task handler
        alt Success
            Processor->>Redis: LREM from active
            Processor->>Redis: DEL task data
        else Failure
            Processor->>Redis: LREM from active
            Processor->>Redis: ZADD to retry queue
        end
    end
```

### 4. Cron Scheduler Implementation

```mermaid
graph TB
    subgraph "Custom Cron System"
        PARSER[Cron Parser<br/>• Parse cron expressions<br/>• Calculate next run<br/>• Timezone support]
        
        REGISTRY[Job Registry<br/>• Store cron jobs<br/>• Job metadata<br/>• Enable/disable]
        
        TICKER[Cron Ticker<br/>• Check every minute<br/>• Find due jobs<br/>• Trigger execution]
        
        EXECUTOR[Job Executor<br/>• Create task payload<br/>• Call client.Enqueue<br/>• Log execution]
    end
    
    PARSER --> REGISTRY
    REGISTRY --> TICKER
    TICKER --> EXECUTOR
    EXECUTOR --> CLIENT[Task Client]
```

## Key Implementation Challenges

### 1. Atomicity & Race Conditions
- Use Redis Lua scripts for atomic operations
- Implement proper locking mechanisms
- Handle concurrent task processing

### 2. Reliability & Recovery
- Worker heartbeat system
- Lease expiry and recovery
- Task retry with exponential backoff
- Dead letter queue for failed tasks

### 3. Scalability
- Worker pool size management
- Queue partitioning strategies
- Connection pooling
- Horizontal scaling support

### 4. Monitoring & Observability
- Task execution metrics
- Queue depth monitoring
- Worker health status
- Performance profiling

## Technology Stack for Implementation

```mermaid
graph LR
    subgraph "Required Technologies"
        GO[Go Language<br/>• Goroutines<br/>• Channels<br/>• Context]
        REDIS_LIB[Redis Client<br/>• go-redis<br/>• Connection pool<br/>• Lua scripts]
        JSON[Serialization<br/>• JSON/Protobuf<br/>• Task encoding]
        HTTP[Web Interface<br/>• Gin/Echo<br/>• REST API<br/>• WebSocket]
        TIME[Time Handling<br/>• Cron parsing<br/>• Timezone support<br/>• Duration calc]
    end
```

This architecture provides a complete blueprint for building a distributed task queue system from scratch, incorporating all the features studied from Asynq: task scheduling, worker pools, cron jobs, monitoring, and reliable delivery mechanisms.

## Questions & Answers

### 1. Can this service accept jobs from multiple apps? How to handle the queues?

**Yes, the service is designed to handle multiple applications seamlessly.**

```mermaid
graph TB
    subgraph "Multiple Applications"
        APP1[E-commerce App<br/>• Order processing<br/>• Email notifications]
        APP2[Analytics App<br/>• Data processing<br/>• Report generation]
        APP3[Auth Service<br/>• Password reset<br/>• User verification]
        APP4[Payment Service<br/>• Transaction processing<br/>• Fraud detection]
    end
    
    subgraph "Queue Management Strategy"
        subgraph "Named Queues"
            Q1[orders queue<br/>High priority<br/>Dedicated workers]
            Q2[emails queue<br/>Medium priority<br/>Bulk processing]
            Q3[analytics queue<br/>Low priority<br/>Background processing]
            Q4[critical queue<br/>Highest priority<br/>Immediate processing]
        end
        
        subgraph "Queue Routing"
            ROUTER[Queue Router<br/>• App identification<br/>• Queue selection<br/>• Load balancing]
        end
    end
    
    subgraph "Redis Queue Storage"
        REDIS_QUEUES[Redis Queues<br/>tasks:orders:pending<br/>tasks:emails:pending<br/>tasks:analytics:pending<br/>tasks:critical:pending]
    end
    
    APP1 --> Q1
    APP1 --> Q2
    APP2 --> Q3
    APP3 --> Q2
    APP4 --> Q4
    
    Q1 --> ROUTER
    Q2 --> ROUTER
    Q3 --> ROUTER
    Q4 --> ROUTER
    
    ROUTER --> REDIS_QUEUES
```

**Queue Handling Strategies:**

1. **Named Queues**: Each application or task type uses dedicated queues
   ```go
   // Different apps enqueue to different queues
   client.Enqueue(task, asynq.Queue("orders"))      // E-commerce
   client.Enqueue(task, asynq.Queue("analytics"))   // Analytics
   client.Enqueue(task, asynq.Queue("emails"))      // Notifications
   ```

2. **Priority-Based Processing**: Different queues can have different priorities
   ```go
   // Worker configuration
   server := asynq.NewServer(redisOpt, asynq.Config{
       Queues: map[string]int{
           "critical":  6,  // Highest priority
           "orders":    4,  // High priority  
           "emails":    2,  // Medium priority
           "analytics": 1,  // Low priority
       },
   })
   ```

3. **Application Isolation**: Separate Redis databases or key prefixes per app
   ```
   app1:tasks:pending    // E-commerce tasks
   app2:tasks:pending    // Analytics tasks
   app3:tasks:pending    // Auth service tasks
   ```

4. **Multi-tenancy Support**: App identification in task metadata
   ```go
   type TaskPayload struct {
       AppID    string `json:"app_id"`
       Data     string `json:"data"`
       TenantID string `json:"tenant_id"`
   }
   ```

### 2. Can this service be scaled up?

**Yes, the service supports multiple scaling strategies:**

```mermaid
graph TB
    subgraph "Horizontal Scaling Architecture"
        subgraph "Multiple Worker Instances"
            W1[Worker Server 1<br/>• 10 goroutines<br/>• handles orders, emails]
            W2[Worker Server 2<br/>• 15 goroutines<br/>• handles analytics]
            W3[Worker Server 3<br/>• 20 goroutines<br/>• handles critical tasks]
            WN[Worker Server N<br/>• Auto-scaling<br/>• Dynamic allocation]
        end
        
        subgraph "Redis Cluster"
            R1[Redis Node 1<br/>• Primary<br/>• High priority queues]
            R2[Redis Node 2<br/>• Primary<br/>• Medium priority queues]
            R3[Redis Node 3<br/>• Primary<br/>• Background queues]
            RS1[Redis Slave 1]
            RS2[Redis Slave 2]
        end
        
        subgraph "Load Balancer"
            LB[Load Balancer<br/>• Health checks<br/>• Queue distribution<br/>• Failover handling]
        end
    end
    
    subgraph "Client Applications"
        CLIENTS[Multiple Apps<br/>• Auto-discovery<br/>• Connection pooling]
    end
    
    CLIENTS --> LB
    LB --> R1
    LB --> R2
    LB --> R3
    
    W1 --> R1
    W1 --> R2
    W2 --> R2
    W2 --> R3
    W3 --> R1
    WN --> R3
    
    R1 --> RS1
    R2 --> RS2
```

**Scaling Strategies:**

1. **Horizontal Worker Scaling**
   ```bash
   # Run multiple worker instances
   ./worker-server --queues=critical,orders --concurrency=20
   ./worker-server --queues=emails,analytics --concurrency=15
   ./worker-server --queues=analytics --concurrency=10
   ```

2. **Auto-scaling Based on Queue Depth**
   ```go
   type AutoScaler struct {
       minWorkers int
       maxWorkers int
       scaleThreshold int
   }
   
   func (as *AutoScaler) ScaleWorkers(queueDepth int) {
       if queueDepth > as.scaleThreshold {
           // Scale up: launch new worker instances
           as.launchWorkerInstance()
       } else if queueDepth < as.scaleThreshold/2 {
           // Scale down: terminate idle workers
           as.terminateWorkerInstance()
       }
   }
   ```

3. **Redis Clustering for High Availability**
   ```
   Redis Sentinel Setup:
   - Master-Slave replication
   - Automatic failover
   - Read scaling with slaves
   
   Redis Cluster Setup:
   - Data sharding across nodes
   - Horizontal partitioning
   - Built-in redundancy
   ```

4. **Queue Partitioning**
   ```go
   // Partition queues by hash
   func getQueueName(taskType string, partitions int) string {
       hash := hash(taskType) % partitions
       return fmt.Sprintf("%s_partition_%d", taskType, hash)
   }
   ```

5. **Container Orchestration**
   ```yaml
   # Kubernetes deployment
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: task-workers
   spec:
     replicas: 5  # Scale based on load
     template:
       spec:
         containers:
         - name: worker
           image: task-worker:latest
           resources:
             requests:
               cpu: 100m
               memory: 128Mi
             limits:
               cpu: 500m
               memory: 512Mi
   ```

**Scaling Metrics to Monitor:**
- Queue depth per queue type
- Task processing rate
- Worker CPU/Memory utilization  
- Redis connection pool usage
- Task failure rates
- Response time percentiles

**Auto-scaling Triggers:**
- Queue depth > threshold → Scale up workers
- Average CPU > 70% → Add worker instances  
- Task processing rate < threshold → Scale down
- Redis memory usage > 80% → Add Redis nodes

The architecture supports both vertical scaling (more powerful instances) and horizontal scaling (more instances), making it suitable for high-throughput production environments.
