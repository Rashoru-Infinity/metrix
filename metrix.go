package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gocolly/colly"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type NodeMetrics struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Metadata   struct {
	} `json:"metadata"`
	Items []struct {
		Metadata struct {
			Name              string    `json:"name"`
			CreationTimestamp time.Time `json:"creationTimestamp"`
			Labels            struct {
				BetaKubernetesIoArch string `json:"beta.kubernetes.io/arch"`
				BetaKubernetesIoOs   string `json:"beta.kubernetes.io/os"`
				KubernetesIoArch     string `json:"kubernetes.io/arch"`
				KubernetesIoHostname string `json:"kubernetes.io/hostname"`
				KubernetesIoOs       string `json:"kubernetes.io/os"`
			} `json:"labels"`
		} `json:"metadata"`
		Timestamp time.Time `json:"timestamp"`
		Window    string    `json:"window"`
		Usage     struct {
			CPU    string `json:"cpu"`
			Memory string `json:"memory"`
		} `json:"usage"`
	} `json:"items"`
}

type PodMetrics struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Metadata   struct {
	} `json:"metadata"`
	Items []struct {
		Metadata struct {
			Name              string    `json:"name"`
			Namespace         string    `json:"namespace"`
			CreationTimestamp time.Time `json:"creationTimestamp"`
			Labels            struct {
				AppKubernetesIoInstance  string `json:"app.kubernetes.io/instance"`
				AppKubernetesIoManagedBy string `json:"app.kubernetes.io/managed-by"`
				AppKubernetesIoName      string `json:"app.kubernetes.io/name"`
				AppKubernetesIoPartOf    string `json:"app.kubernetes.io/part-of"`
				PodTemplateHash          string `json:"pod-template-hash"`
				Product                  string `json:"product"`
				Profile                  string `json:"profile"`
			} `json:"labels"`
		} `json:"metadata,omitempty"`
		Timestamp  time.Time `json:"timestamp"`
		Window     string    `json:"window"`
		Containers []struct {
			Name  string `json:"name"`
			Usage struct {
				CPU    string `json:"cpu"`
				Memory string `json:"memory"`
			} `json:"usage"`
		} `json:"containers"`
	} `json:"items"`
}

func GetJSON(url string, jsonContent *string, token string) {
	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Authorization", "Bearer " + token)
	})
	c.OnResponse(func(r *colly.Response) {
		*jsonContent = string(r.Body)
	})
	c.OnError(func(r *colly.Response, err error) {
		fmt.Fprintln(os.Stderr, r.Body)
	})
	c.Visit(url)
}

func main() {
	var (
		interval = flag.Duration("interval", 1000 * time.Millisecond, "Interval for collecting metrics data.")
		nodeURL = flag.String("node", "https://kubernetes.default.svc/apis/metrics.k8s.io/v1beta1/nodes/", "URL for collecting nodes metrics.")
		podURL = flag.String("pod", "https://kubernetes.default.svc/apis/metrics.k8s.io/v1beta1/pods/", "URL for collecting pods metrics.")
		tokenFile = flag.String("token", "/var/run/secrets/kubernetes.io/serviceaccount/token", "Bearer token file for authorization.")
	)
	bytes, err := os.ReadFile(*tokenFile)
	if err != nil {
		panic(err)
	}
	token := string(bytes)
	var wg sync.WaitGroup
	flag.Parse()
	for {
		/* collect node metrics */
		wg.Add(1)
		go func(url string, token string){
			defer wg.Done()
			var metrics string
			GetJSON(url, &metrics, token)
			dec := json.NewDecoder(strings.NewReader(metrics))
			for {
				var nm NodeMetrics
				if err := dec.Decode(&nm); err == io.EOF {
					break
				} else if err != nil {
					log.Fatal(err)
				}
				for i := 0;i < len(nm.Items);i++ {
					fmt.Printf("%s\n", nm.Items[i].Metadata.Name)
					cpusrc :=  nm.Items[i].Usage.CPU
					cpumtx, err := strconv.ParseInt(cpusrc[:len(cpusrc) - 1], 10, 64)
					if err == nil {
						fmt.Printf("%d(ns)", cpumtx)
					} else {
						log.Fatal(err)
					}
					fmt.Printf("%s\n", nm.Items[i].Usage.CPU)
					memsrc := nm.Items[i].Usage.Memory
					memmtx, err := strconv.ParseInt(memsrc[:len(memsrc) - 2], 10, 64)
					if err == nil {
						fmt.Printf("%d(ki)\n", memmtx)
					} else {
						log.Fatal(err)
					}
					fmt.Printf("%s\n", nm.Items[i].Usage.Memory)
				}
			}
		}(*nodeURL, token)
		/* collect pod metrics */
		wg.Add(1)
		go func(url string, token string){
			defer wg.Done()
			var metrics string
			GetJSON(url, &metrics, token)
			dec := json.NewDecoder(strings.NewReader(metrics))
			for {
				var pm PodMetrics
				if err := dec.Decode(&pm); err == io.EOF {
					break
				} else if err != nil {
					log.Fatal(err)
				}
				for i := 0;i < len(pm.Items);i++ {
					fmt.Printf("%s\n", pm.Items[i].Metadata.Namespace)
					fmt.Printf("%s\n", pm.Items[i].Metadata.Name)
					for j := 0;j < len(pm.Items[i].Containers);j++ {
						cpusrc := pm.Items[i].Containers[j].Usage.CPU
						cpumtx, err := strconv.ParseInt(cpusrc[:len(cpusrc) - 1], 10, 64)
						if err == nil {
							fmt.Printf("%d(ns)\n", cpumtx)
						} else {
							log.Fatal(err)
						}
						memsrc := pm.Items[i].Containers[j].Usage.Memory
						memmtx, err := strconv.ParseInt(memsrc[:len(memsrc) - 2], 10, 64)
						if err == nil {
							fmt.Printf("%d(ns)\n", memmtx)
						} else {
							log.Fatal(err)
						}
					}
				}
			}
		}(*podURL, token)
		wg.Wait()
		time.Sleep(*interval)
	}
}