package event

import (
	"context"
	"os"
	"reflect"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/olivere/elastic.v5"
)

const (
	Index = "events"
	DocumentType = "json"
	Version = "0.7"
)

type DocStore struct {
	cluster     string
	rack        string
	host        string
	component   string
	env         string
	client      *elastic.Client
}

type Event struct {
	Version     string
	Cluster     string
	Rack        string
	Host        string
	Component   string
	Type        string
	Id          string
	Name        string
	Ref         string
	Msg         interface{}
	Created     time.Time
}

const (
	KeyVersion = "Version"
	KeyCluster = "Cluster"
	KeyRack = "Rack"
	KeyHost = "Host"
	KeyComponent = "Component"
	KeyType = "Type"
	KeyId = "Id"
	KeyName = "Name"
	KeyEvent = "Msg"
	KeyCreate = "Created"

	TypeService = "Service"
	TypeTask = "Task"
	TypeVolume = "Volume"
	TypeNetwork = "Network"
	TypeEtc = "Etc"
)

const mapping = `{
	"mappings":{
		"event":{
			"properties":{
				"cluster":{
					"type":"keyword"
				},
				"rack":{
					"type":"keyword"
				},
				"host":{
					"type":"keyword"
				},
				"component":{
					"type":"keyword"
				},
				"msg":{
					"type":"object"
				},
				"created":{
					"type":"date"
				}
			}
		}
	}
}`

func NewEventStoreByEnv(url string) (*DocStore, error) {
	if url == "" {
		err := errors.Errorf("new: URL should not be nil")
		logrus.Error(err)
		return nil, err
	}

	client, err := elastic.NewClient(elastic.SetURL(url))
	if err != nil {
		logrus.Error("new|client: ", err)
		return nil, err
	}

	ds := &DocStore {
		cluster: "",
		component: "",
		rack: "",
		host: "",
		client: client,
	}

	exists, err := ds.exists()
	if err != nil {
		logrus.Error("new|exists: ", err)
		return nil, err
	}

	if !exists {
		err := ds.create()
		if err != nil {
			logrus.Error("new|create: ", err)
			return nil, err
		}
	}

	return ds, err
}

func NewEventStore(cluster, rack, host, component, url string) (*DocStore, error) {
	if host == "" {
		name, err := os.Hostname()
		if err != nil {
			err := errors.Errorf("new: could not get hostname")
			logrus.Error(err)
			return nil, err
		}
		host = name
	}

	if cluster == "" || rack == "" {
		var err error
		cluster, rack, err = getClusterAndRack(host)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
	}

	client, err := elastic.NewClient(elastic.SetURL(url))
	if err != nil {
		logrus.Error("new|client: ", err)
		return nil, err
	}

	ds := &DocStore {
		cluster: cluster,
		component: component,
		rack: rack,
		host: host,
		client: client,
	}

	exists, err := ds.exists()
	if err != nil {
		logrus.Error("new|exists: ", err)
		return nil, err
	}

	if !exists {
		err := ds.create()
		if err != nil {
			logrus.Error("new|create: ", err)
			return nil, err
		}
	}

	return ds, err
}

func (d *DocStore) UpdateCluster(cluster, rack, host, component string) {
	d.cluster = cluster
	d.rack = rack
	d.host = host
	d.component = component
}

func (d *DocStore) insert(event *Event) error {
	resp, err := d.client.Index().Index(Index).Type(DocumentType).BodyJson(event).Do(context.Background())
	if err != nil {
		logrus.Error("insert: ", err)
		return err
	} else {
		logrus.Debug("insert: ", resp)
	}

	_, err = d.client.Flush().Index(Index).Do(context.Background())
	if err != nil {
		return err
	}

	return err
}

func (d *DocStore) InsertWithService(doc interface{}, id, name string) error {
	ev := Event {
		Version: Version,
		Cluster: d.cluster,
		Rack: d.rack,
		Host: d.host,
		Component: d.component,
		Type: TypeService,
		Id: id,
		Name: name,
		Msg: doc,
		Created: time.Now(),
	}

	return d.insert(&ev)
}

func (d *DocStore) InsertWithTask(doc interface{}, id, name, ref string) error {
	ev := Event {
		Version: Version,
		Cluster: d.cluster,
		Rack: d.rack,
		Host: d.host,
		Component: d.component,
		Type: TypeTask,
		Id: id,
		Name: name,
		Ref: ref,
		Msg: doc,
		Created: time.Now(),
	}

	return d.insert(&ev)
}

func (d *DocStore) Insert(doc interface{}) error {
	ev := Event {
		Version: Version,
		Cluster: d.cluster,
		Rack: d.rack,
		Host: d.host,
		Component: d.component,
		Type: TypeEtc,
		Id: "",
		Name: "",
		Msg: doc,
		Created: time.Now(),
	}

	return d.insert(&ev)
}

func (d *DocStore) SearchByRawString(query string, from, size int) ([]*Event, error) {
	stringQuery := elastic.NewRawStringQuery(query)
	searchResult, err := d.client.Search().Index(Index).Query(stringQuery).From(from).Size(size).Sort(KeyCreate, true).Pretty(true).Do(context.Background())
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	logrus.Debugf("Query took %d milliseconds", searchResult.TookInMillis)
	logrus.Debugf("Query result hits: %v", searchResult.Hits.TotalHits)

	var ev Event
	var rs []*Event
	for _, item := range searchResult.Each(reflect.TypeOf(ev)) {
		if t, ok := item.(Event); ok {
			//logrus.Debugf("doc: %v", t)
			rs = append(rs, &t)
		}
	}

	return rs, nil
}

func (d *DocStore) SearchByQuery(query elastic.Query, from, size int) ([]*Event, error) {
	searchResult, err := d.client.Search().Index(Index).Query(query).From(from).Size(size).Sort(KeyCreate, true).Pretty(true).Do(context.Background())
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	logrus.Debugf("Query took %d milliseconds", searchResult.TookInMillis)
	logrus.Debugf("Query result hits: %v", searchResult.Hits.TotalHits)

	var ev Event
	var rs []*Event
	for _, item := range searchResult.Each(reflect.TypeOf(ev)) {
		if t, ok := item.(Event); ok {
			//logrus.Debugf("doc: %v", t)
			rs = append(rs, &t)
		}
	}

	return rs, nil
}

func (d DocStore) SearchByMap(term map[string]string, match map[string]string, from, size int) ([]*Event, error) {
	var leafs []elastic.Query
	for k, v := range term {
		leafs = append(leafs, elastic.NewTermQuery(k, v))
	}
	for k, v := range match {
		leafs = append(leafs, elastic.NewMatchQuery(k, v))
	}

	return d.SearchByQuery(elastic.NewBoolQuery().Must(leafs...), from, size)
}

func (d *DocStore) SearchCluster(cluster string) ([]*Event, error) {
	termQuery := elastic.NewTermQuery(KeyCluster, cluster)
	return d.SearchByQuery(termQuery, 0, 100)
}

func (d *DocStore) SearchRack(rack, cluster string) ([]*Event, error) {
	var terms []elastic.Query
	if cluster != "" {
		terms = append(terms, elastic.NewTermQuery(KeyCluster, cluster))
	}
	terms = append(terms, elastic.NewTermQuery(KeyRack, rack))

	return d.SearchByQuery(elastic.NewBoolQuery().Must(terms...), 0, 100)
}

func (d *DocStore) SearchHost(host, cluster, rack string) ([]*Event, error) {
	var terms []elastic.Query
	if cluster != "" {
		terms = append(terms, elastic.NewTermQuery(KeyCluster, cluster))
	}
	if rack != "" {
		terms = append(terms, elastic.NewTermQuery(KeyRack, rack))
	}
	terms = append(terms, elastic.NewTermQuery(KeyHost, host))

	return d.SearchByQuery(elastic.NewBoolQuery().Must(terms...), 0, 100)
}

func (d *DocStore) SearchComponent(component, cluster, rack, host string) ([]*Event, error) {
	var terms []elastic.Query
	if cluster != "" {
		terms = append(terms, elastic.NewTermQuery(KeyCluster, cluster))
	}
	if rack != "" {
		terms = append(terms, elastic.NewTermQuery(KeyRack, rack))
	}
	if host != "" {
		terms = append(terms, elastic.NewTermQuery(KeyHost, host))
	}
	terms = append(terms, elastic.NewTermQuery(KeyComponent, component))

	return d.SearchByQuery(elastic.NewBoolQuery().Must(terms...), 0, 100)
}

func (d *DocStore) SearchService(serviceId, serviceName, cluster string) ([]*Event, error) {
	var terms []elastic.Query
	if cluster != "" {
		terms = append(terms, elastic.NewTermQuery(KeyCluster, cluster))
	}
	if serviceId != "" {
		terms = append(terms, elastic.NewTermQuery(KeyId, serviceId))
	}
	if serviceName != "" {
		terms = append(terms, elastic.NewTermQuery(KeyName, serviceName))
	}
	terms = append(terms, elastic.NewTermQuery(KeyType, TypeService))

	return d.SearchByQuery(elastic.NewBoolQuery().Must(terms...), 0, 100)
}

func (d *DocStore) SearchTask(taskId, taskName, cluster string) ([]*Event, error) {
	var terms []elastic.Query
	if cluster != "" {
		terms = append(terms, elastic.NewTermQuery(KeyCluster, cluster))
	}
	if taskId != "" {
		terms = append(terms, elastic.NewTermQuery(KeyId, taskId))
	}
	if taskName != "" {
		terms = append(terms, elastic.NewTermQuery(KeyName, taskName))
	}
	terms = append(terms, elastic.NewTermQuery(KeyType, TypeTask))

	return d.SearchByQuery(elastic.NewBoolQuery().Must(terms...), 0, 100)
}

func (d *DocStore) Reset() error {
	err := d.delete()
	if err != nil {
		return err
	} else {
		err = d.create()
	}

	return err
}

func (d *DocStore) create() error {
	create, err := d.client.CreateIndex(Index).BodyString(mapping).Do(context.Background())
	if err != nil {
		logrus.Error("create: ", err)
		return err
	}

	if !create.Acknowledged {
		logrus.Error("create|acknowledged: ", err)
		return errors.New("create index not acknowledged")
	}

	return nil
}

func (d *DocStore) delete() error {
	resp, err := d.client.DeleteIndex(Index).Do(context.Background())
	if err != nil {
		logrus.Error("delete: ", err)
		return err
	} else {
		logrus.Debug("delete: ", resp)
	}

	return nil
}

func (d *DocStore) exists() (bool, error) {
	exists, err := d.client.IndexExists(Index).Do(context.Background())
	if err != nil {
		logrus.Error("exists: ", err)
		return false, err
	}

	return exists, err
}

func (d *DocStore) setInfoByHost() error {
	return nil
}
