package event

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"strings"
	"testing"
	"time"
)

type Sample struct {
	Nick        string
}

type Actor struct {
	Name        string
	Attribute   map[string]string
}

const ElasticSearch = "http://elastic:JcoJfjJuOggYSRSY@shev-dev.ppr2.io.navercorp.com:10200"

var clusters = []string { "red", "blue", "green" }
var racks = []string { "r01", "r02" }
var hosts = []string { "laptop", "desktop", "tablet", "cellphone" }
var components = []string { "windows", "mac", "linux" }
//var contents = []Sample { { Nick:"Milky Way" }, { Nick:"Little Girl" }, { Nick:"Fat Baby" } }
var contents = []Actor { { Name: "albam", Attribute: map[string]string{ "aaa.bbb.ccc":"value", "111.222.333":"value" } } }
var services = []string { "cafe", "blog", "search" }
var tasks = []string { "cafe.1", "cafe.2", "blog.1", "blog.2", "blog.3", "search.1" }

func load(ds *DocStore, cluster, rack, host, component string) {
	ds.UpdateCluster(cluster, rack, host, component)

	for _, sr := range services {
		for _, co := range contents {
			ds.InsertWithService(co, "", sr)
		}
	}

	for _, ta := range tasks {
		for _, sr := range services {
			if ! strings.HasPrefix(ta, sr) {
				continue
			}

			for _, co := range contents {
				ds.InsertWithTask(co,"", ta, sr)
			}
		}
	}
}

func TestInit(t *testing.T) {
	b := time.Now()
	ds, e := NewEventStoreByEnv(ElasticSearch)
	if e != nil {
		panic(e)
	}
	fmt.Println("NewEventStore elapsed: ", time.Now().Sub(b))

	b = time.Now()
	for _, cl := range clusters {
		for _, ra := range racks {
			for _, ho := range hosts {
				for _, co := range components {
					load(ds, cl, ra, ho, co)
				}
			}
		}
	}
	fmt.Println("TestInit elapsed: ", time.Now().Sub(b))
}

func TestReset(t *testing.T) {
	ds, e := NewEventStoreByEnv(ElasticSearch)
	if e != nil {
		panic(e)
	}

	ds.Reset()
}

func printEvents(es []*Event, e error) {
	if e != nil {
		return
	}

	for i, ev := range es {
		//vvv, _ := json.Marshal(ev)
		//fmt.Printf("%s", vvv)
		fmt.Printf("[%04d] %+v\n", i, ev)
	}
}

func TestDocStore_Search(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	ds, e := NewEventStoreByEnv(ElasticSearch)
	if e != nil {
		panic(e)
	}

	printEvents(ds.SearchCluster("red"))
	printEvents(ds.SearchRack("r01", "red"))
	printEvents(ds.SearchHost("laptop", "red", "r01"))
	printEvents(ds.SearchComponent("windows", "red", "r01", "laptop"))
}

func TestDocStore_SearchByMap(t *testing.T) {
	ds, e := NewEventStoreByEnv(ElasticSearch)
	if e != nil {
		panic(e)
	}

	var term = map[string]string {
		KeyCluster: "red",
		KeyComponent: "linux",
		KeyType: TypeService,
		KeyName: "blog",
	}
	var match = map[string]string {
		KeyEvent: "Baby",
	}
	printEvents(ds.SearchByMap(term, match,0, 100))
}

func TestDocStore_SearchByRawString(t *testing.T) {
	ds, e := NewEventStoreByEnv(ElasticSearch)
	if e != nil {
		panic(e)
	}

	var query = `
{
  "bool": {
    "must": [
      { "term": { "Cluster": "red" } },
      { "term": { "Component": "linux" } }
    ]
  }
}
`
	printEvents(ds.SearchByRawString(query,0, 100))
}
