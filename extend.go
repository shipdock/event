package event

import (
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"strings"
)

var addresses = []string {
	"127.0.0.1:8500",
}

func getClusterAndRack(hostname string) (string, string, error) {
	key, err := getHostPath(hostname)
	if err != nil {
		logrus.Errorf("getHostPath: %v", err)
		return "", "", err
	}

	s := strings.Split(key, "/")
	return s[2], s[4], nil
}

func printCluster(hostname string) error {
	key, err := getHostPath(hostname)
	if err != nil {
		logrus.Errorf("print: %v", err)
		return err
	}

	s := strings.Split(key, "/")

	logrus.Infof("find host('%v'): '%v' cluster, '%v' rack", hostname, s[2], s[4])
	return nil
}

func getConsulClient() (*api.Client, error) {
	config := api.DefaultConfig()
	config.HttpClient = http.DefaultClient
	config.Address = addresses[0]
	config.Scheme = "http"

	// Creates a new client
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getHostPath(hostname string) (string, error) {
	client, err := getConsulClient()
	if err != nil {
		logrus.Errorf("consul new client: %v", err)
		return "", err
	}

	path, err := findHost(client, hostname)
	if err != nil {
		return "", err
	}

	return path, nil
}

func findHost(client *api.Client, hostname string) (string, error) {
	root := "shipdock/clusters/"
	clusters, _, err := client.KV().Keys(root, "/", nil)
	if err != nil {
		logrus.Errorf("kv.List(): '%v' - %v", root, err)
		return "", err
	}

	for _, key := range clusters {
		cluster := fmt.Sprintf("%vracks/", key)
		racks, _, err := client.KV().Keys(cluster, "/", nil)
		if err != nil {
			logrus.Errorf("kv.List(): '%v' - %v", cluster, err)
			time.Sleep(1 * time.Second)
			continue
		}

		for _, key := range racks {
			rack := fmt.Sprintf("%vhosts/", key)
			hosts, _, err := client.KV().Keys(rack, "/", nil)
			if err != nil {
				logrus.Errorf("kv.List(): '%v' - %v", rack, err)
				break
			}

			for _, key := range hosts {
				if strings.Count(key, hostname) > 0 {
					logrus.Debugf("find path: '%v/%v'", rack, key)
					return path.Join(rack, key), nil
				}
			}
		}
	}

	return "", errors.Errorf("could not find host: '%v'", hostname)
}
