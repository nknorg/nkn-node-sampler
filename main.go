package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type rpcSuccessorResponse struct {
	Result []string `json:"result"`
}

type rpcChordRingInfoResponse struct {
	Result struct {
		LocalNode struct {
			ID                string `json:"id"`
			RelayMessageCount uint64 `json:"relayMessageCount"`
			Uptime            int    `json:"uptime"`
		} `json:"localNode"`
		Successors []struct {
			Addr string `json:"addr"`
			ID   string `json:"id"`
		} `json:"successors"`
		Predecessors []struct {
			Addr string `json:"addr"`
			ID   string `json:"id"`
		} `json:"predecessors"`
	} `json:"result"`
}

var (
	// Version string
	Version string
)

var (
	totalSpace *big.Int = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
)

func makeRPCRequest(url string, method string, params interface{}) ([]byte, error) {
	requestBody := struct {
		Method string      `json:"method"`
		Params interface{} `json:"params"`
	}{
		Method: method,
		Params: params,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   time.Second * 5,
				KeepAlive: time.Second * 30,
			}).DialContext,
			TLSHandshakeTimeout:   time.Second * 5,
			ResponseHeaderTimeout: time.Second * 5,
			ExpectContinueTimeout: time.Second * 1,
		},
	}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func tcpAddrToRPCAddr(tcpAddr string) (string, error) {
	parsedURL, err := url.Parse(tcpAddr)
	if err != nil {
		return "", err
	}
	host := strings.Split(parsedURL.Host, ":")[0]
	return fmt.Sprintf("http://%s:30003", host), nil
}

func sampleNodes(startKey *big.Int, initialRPCAddress string, n int) (int, uint64, int, *big.Int, error) {
	findSuccessorReq := struct {
		Key string `json:"key"`
	}{
		Key: startKey.Text(16),
	}

	respBody, err := makeRPCRequest(initialRPCAddress, "findsuccessoraddrs", findSuccessorReq)
	if err != nil {
		return 0, 0, 0, nil, fmt.Errorf("Error making RPC request: %v", err)
	}

	var successorResp rpcSuccessorResponse
	if err := json.Unmarshal(respBody, &successorResp); err != nil || len(successorResp.Result) == 0 {
		return 0, 0, 0, nil, fmt.Errorf("Error parsing findsuccessoraddrs response: %v", err)
	}

	rpcAddresses := make([]string, 1)
	if len(successorResp.Result) == 0 {
		return 0, 0, 0, nil, fmt.Errorf("Found no successors")
	}

	firstAddr := successorResp.Result[0]
	rpcAddresses[0], err = tcpAddrToRPCAddr(firstAddr)
	if err != nil {
		return 0, 0, 0, nil, fmt.Errorf("Error parsing URL from successor address: %v", err)
	}

	nodesVisited := 1
	lastID := startKey.Text(16)
	var totalRelay uint64
	totalUptime := 0
	for i := 0; i <= n; i++ {
		var responseBody []byte
		for _, rpcAddress := range rpcAddresses {
			responseBody, err = makeRPCRequest(rpcAddress, "getchordringinfo", struct{}{})
			if err == nil {
				break
			}
		}
		if err != nil {
			log.Println("Error getting chord ring info:", err)
			break
		}

		var chordRingResp rpcChordRingInfoResponse
		if err := json.Unmarshal(responseBody, &chordRingResp); err != nil {
			log.Println("Error unmarshalling chord ring info response:", err)
			break
		}

		successors := chordRingResp.Result.Successors
		predecessors := chordRingResp.Result.Predecessors
		if i > 0 {
			found := false
			for j, pred := range predecessors {
				if pred.ID == lastID {
					nodesVisited += j + 1
					found = true
					break
				}
			}
			if !found {
				log.Println("Prev ID not found in predecessors")
				break
			}
		}

		lastID = chordRingResp.Result.LocalNode.ID
		totalRelay += chordRingResp.Result.LocalNode.RelayMessageCount
		totalUptime += chordRingResp.Result.LocalNode.Uptime

		if len(successors) < 1 {
			log.Println("Not enough successors")
			break
		}

		rpcAddresses = make([]string, 0)
		mid := len(successors) - int(2*math.Sqrt(float64(len(successors))))
		for index := mid - 1; index <= mid+1; index++ {
			if index > 0 && index < len(successors) {
				rpcAddress, err := tcpAddrToRPCAddr(successors[index].Addr)
				if err != nil {
					log.Println("Error parsing URL from successor address:", err)
					continue
				}
				rpcAddresses = append(rpcAddresses, rpcAddress)
			}
		}
		if len(rpcAddresses) == 0 {
			break
		}
	}

	var totalArea *big.Int
	lastKey := new(big.Int)
	lastKey.SetString(lastID, 16)

	if startKey.Cmp(lastKey) > 0 {
		totalArea = new(big.Int).Sub(totalSpace, startKey)
		totalArea.Add(totalArea, lastKey)
	} else {
		totalArea = new(big.Int).Sub(lastKey, startKey)
	}

	return nodesVisited, totalRelay, totalUptime, totalArea, nil
}

func main() {
	timeStart := time.Now()

	var (
		rpcAddr string
		m, n    int
		jsFmt   bool
		version bool
	)

	flag.StringVar(&rpcAddr, "rpc", "http://seed.nkn.org:30003", "Initial RPC address in the form ip:port")
	flag.IntVar(&m, "m", 8, "Number of concurrent goroutines")
	flag.IntVar(&n, "n", 8, "Number of steps to repeat")
	flag.BoolVar(&jsFmt, "json", false, "Print version")
	flag.BoolVar(&version, "version", false, "Print version")
	flag.Parse()

	if version {
		fmt.Println(Version)
		os.Exit(0)
	}

	if len(rpcAddr) == 0 {
		log.Fatal("RPC address is required")
	}

	wg := &sync.WaitGroup{}
	nodeCountChan := make(chan int, m)
	areaChan := make(chan *big.Int, m)
	relayCountChan := make(chan uint64, m)
	uptimeChan := make(chan int, m)

	S, err := rand.Int(rand.Reader, totalSpace)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < m; i++ {
		wg.Add(1)
		offset := new(big.Int).Mul(totalSpace, big.NewInt(int64(i)))
		offset.Div(offset, big.NewInt(int64(m)))
		startKey := new(big.Int).Add(S, offset)
		startKey.Mod(startKey, totalSpace)

		go func() {
			defer wg.Done()
			nodeCount, relayCount, uptime, area, err := sampleNodes(startKey, rpcAddr, n)
			if err != nil {
				log.Println(err)
			}
			nodeCountChan <- nodeCount
			relayCountChan <- relayCount
			uptimeChan <- uptime
			areaChan <- area
		}()
	}

	wg.Wait()
	close(nodeCountChan)
	close(areaChan)
	close(relayCountChan)
	close(uptimeChan)

	totalNodesVisited := 0
	var totalRelay uint64
	var totalUptime int
	totalArea := big.NewInt(0)
	for nodesVisited := range nodeCountChan {
		totalNodesVisited += nodesVisited
	}
	for relayCount := range relayCountChan {
		totalRelay += relayCount
	}
	for uptime := range uptimeChan {
		totalUptime += uptime
	}
	for area := range areaChan {
		if area != nil {
			totalArea.Add(totalArea, area)
		}
	}

	if totalArea.Sign() == 0 {
		log.Fatal("Error: Total area covered is zero, cannot estimate total number of nodes.")
	}
	estimatedTotalNodes := new(big.Int).Div(new(big.Int).Mul(big.NewInt(int64(totalNodesVisited)), totalSpace), totalArea).Int64()
	uncertainty := new(big.Int).Div(new(big.Int).Mul(big.NewInt(int64(math.Sqrt(float64(totalNodesVisited)))), totalSpace), totalArea).Int64()
	estimatedRelayPerSecond := float64(totalRelay) / float64(totalUptime) * float64(estimatedTotalNodes) / (math.Log2(float64(estimatedTotalNodes)) / 2)

	if jsFmt {
		jsStr, err := json.Marshal(map[string]float64{
			"visited":     float64(totalNodesVisited),
			"covered":     100 * float64(totalNodesVisited) / float64(estimatedTotalNodes),
			"Estimated":   float64(estimatedTotalNodes),
			"uncertainty": float64(uncertainty),
			"relayPS":     estimatedRelayPerSecond,
			"time":        float64(time.Since(timeStart)),
		})
		if err != nil {
			log.Fatalln("json formatter error")
		}
		fmt.Println(string(jsStr))
	} else {
		log.Printf("Total nodes visited: %d\n", totalNodesVisited)
		log.Printf("Total area covered: %.2f%%\n", 100*float64(totalNodesVisited)/float64(estimatedTotalNodes))
		log.Printf("Estimated total number of nodes in the network: %d +- %d\n", estimatedTotalNodes, uncertainty)
		log.Printf("Estimated network relay per second: %.0f\n", estimatedRelayPerSecond)
		log.Printf("Time used: %v\n", time.Since(timeStart))
	}
}
