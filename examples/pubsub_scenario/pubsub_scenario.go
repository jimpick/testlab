package main

import (
	"context"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p-daemon/p2pclient"
	p2pd_pb "github.com/libp2p/go-libp2p-daemon/pb"
	"github.com/libp2p/testlab/scenario"
	"github.com/sirupsen/logrus"
)

const topic = "load-test"

func subscribeReceivers(clients []*p2pclient.Client) {
	for _, client := range clients {
		msgs, err := client.Subscribe(context.Background(), topic)
		if err != nil {
			logrus.Errorf("error subscribing: %s", err)
		}
		id, _, err := client.Identify()
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Infof("subscribed client %s", id)
		go func(msgs <-chan *p2pd_pb.PSMessage) {
			for msg := range msgs {
				logrus.Infof("%s got message on topic: %s", id, msg.TopicIDs[0])
			}
		}(msgs)
	}
}

func main() {
	logrus.SetLevel(logrus.InfoLevel)
	runner, err := scenario.NewScenarioRunner()
	if err != nil {
		logrus.Fatal(err)
	}

	peers, err := runner.Peers()
	if err != nil {
		logrus.Fatal(err)
	}

	if len(peers) < 2 {
		logrus.Fatalf("scenario needs at least 2 peers to run, found %d", len(peers))
	}

	sender := peers[0]
	receivers := peers[1:]

	senderId, senderAddrs, err := sender.Identify()
	if err != nil {
		logrus.Fatal(err)
	}
	for _, receiver := range receivers {
		err := receiver.Connect(senderId, senderAddrs)
		if err != nil {
			logrus.Fatalf("connecting to sender %s", err)
		}
	}
	go subscribeReceivers(receivers)

	for {
		wait := rand.Int63n(500000)
		logrus.Infof("Sending a message in %dms", wait)
		time.Sleep(time.Duration(wait) * time.Millisecond)
		data := make([]byte, rand.Intn(450)+50)
		rand.Read(data)
		err := sender.Publish(topic, data)
		if err != nil {
			logrus.Error(err)
		}
	}
}
