# Shipdock Event Platform
## Event handling library
### 목표
이 라이브러리는 Shipdock 클러스터에서 발생하는 모든 이벤트들을 저장하고 조회하는데 쓰인다.
이벤트를 생성하는 주체(dockerd, gateway, agent, gorb 등)들은 직접 이 라이브러리를 사용하여 이벤트를 저장해야 한다.
최종적인 저장소는 ElasticSearch 이며 각 클러스트의 위상에 따라 저장소를 달리 해야한다.

### 저장 구조
- 이벤트의 저장 구조는 다음과 같다.
```go
type Event struct {
	Version     string
	Cluster     string
	Rack        string
	Host        string
	Component   string
	ServiceId   string
	ServiceName string
	TaskId      string
	TaskName    string
	Msg         interface{}
	Created     time.Time
}
```

- ElasticSearch 저장 예시
```json
{
  "_index": "events",
  "_type": "text",
  "_id": "AWEnRASk5z2Ff-6T7Jjq",
  "_score": 1.0162971,
  "_source": {
    "Version": "0.7",
    "Cluster": "red",
    "Rack": "r01",
    "Host": "laptop",
    "Component": "linux",
    "ServiceId": "",
    "ServiceName": "blog",
    "TaskId": "",
    "TaskName": "blog.2",
    "Msg": """{"Nick":"Fat Baby"}""",
    "Created": "2018-01-24T17:23:44.536458962+09:00"
  }
}
```

### 저장
- 저장하기
```go
    store, error := NewEventStoreByEnv(EnvTest)
    if error != nil {
        panic(error)
    }

    store.UpdateCluster("cluster", "rack", "host", "component")
    store.InsertWithServiceAndTask(event, "serviceID", "serviecName", "taskID", "taskName")
```

- 현재 클러스터 매핑
```go
var PreDefineClusterEnvMap = map[string]string {
	"build": EnvDev,
	"dpd1":  EnvTest,
	"dpd2":  EnvTest,
	"edu":   EnvTest,
	"exp":   EnvTest,
	"ksd1":  EnvTest,
	"pcd1":  EnvDev,
	"pcr1":  EnvReal,
	"play":  EnvTest,
	"ppr1":  EnvReal,
	"ppr2":  EnvReal,
	"ppr3":  EnvReal,
	"test":  EnvDev,
	"pxr1":  EnvExternal,
	"pxr2":  EnvExternal,
}
```

### 조회
- 자세한 내용은 [event_test.go](https://oss.navercorp.com/shipdock/event/blob/master/event_test.go)를 참고 하세요.
