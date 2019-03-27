package rabbitmq

import (
	"encoding/json"
	"fmt"
	"github.com/godaddy-x/jorm/log"
	"github.com/streadway/amqp"
	"sync"
	"time"
)

var (
	pull_mgrs = make(map[string]*PullManager)
)

type PullManager struct {
	conn *amqp.Connection
	pull *PullMQ
}

type PullMQ struct {
	mu   sync.Mutex
	receivers []Receiver
}

func (self *PullManager) InitConfig(input ...AmqpConfig) *PullManager {
	for _, v := range input {
		c, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/", v.Username, v.Password, v.Host, v.Port))
		if err != nil {
			panic("RabbitMQ初始化失败: " + err.Error())
		}
		pull_mgr := &PullManager{
			conn: c,
			pull: &PullMQ{
				receivers: make([]Receiver, 0),
			},
		}
		if len(v.DsName) == 0 {
			v.DsName = MASTER
		}
		pull_mgrs[v.DsName] = pull_mgr
	}
	return self
}

func (self *PullManager) Client(dsname ...string) (*PullManager, error) {
	var ds string
	if len(dsname) > 0 && len(dsname[0]) > 0 {
		ds = dsname[0]
	} else {
		ds = MASTER
	}
	manager := pull_mgrs[ds]
	return manager, nil
}

func (self *PullManager) AddPullReceiver(receivers ...Receiver) {
	for _, v := range receivers {
		self.pull.receivers = append(self.pull.receivers, v)
		go func() {
			self.pull.start(v)
		}()
	}
}

func (self *PullMQ) run(receiver Receiver) {
	wg := receiver.Group()
	wg.Add(1)
	go self.listen(receiver)
	wg.Wait()
	log.Error("消费通道意外关闭,需要重新连接")
	receiver.Channel().Close()
}

func (self *PullMQ) start(receiver Receiver) {
	for {
		self.run(receiver)
		time.Sleep(3 * time.Second)
	}
}

func (self *PullMQ) listen(receiver Receiver) {
	defer receiver.Group().Done()
	fmt.Sprintf("队列[%s - %s]消费服务启动成功...", receiver.ExchangeName(), receiver.QueueName())
	channel := receiver.Channel()
	exchange := receiver.ExchangeName()
	queue := receiver.QueueName()
	//testSend(exchange, queue)
	if err := self.prepareExchange(channel, exchange); err != nil {
		receiver.OnError(fmt.Errorf("初始化交换机 [%s] 失败: %s", receiver.ExchangeName(), err.Error()))
		return
	}
	if err := self.prepareQueue(channel, exchange, queue); err != nil {
		receiver.OnError(fmt.Errorf("绑定队列 [%s] 到交换机失败: %s", receiver.QueueName(), err.Error()))
		return
	}
	channel.Qos(1, 0, true)
	if msgs, err := channel.Consume(queue, "", false, false, false, false, nil); err != nil {
		receiver.OnError(fmt.Errorf("获取队列 %s 的消费通道失败: %s", queue, err.Error()))
	} else {
		for msg := range msgs {
			for !receiver.OnReceive(msg.Body) {
				fmt.Println("receiver 数据处理失败，将要重试")
				time.Sleep(1 * time.Second)
			}
			msg.Ack(false)
		}
	}
}

func (self *PullMQ) prepareExchange(channel *amqp.Channel, exchange string) error {
	return channel.ExchangeDeclare(exchange, "direct", true, false, false, false, nil)
}

func (self *PullMQ) prepareQueue(channel *amqp.Channel, exchange, queue string) error {
	if _, err := channel.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := channel.QueueBind(queue, queue, exchange, false, nil); err != nil {
		return err
	}
	return nil
}

func testSend(exchange, queue string) {
	go func() {
		time.Sleep(3 * time.Second)
		for i := 0; i < 10; i++ {
			cli, _ := new(PublishManager).Client()
			v := map[string]interface{}{"test": 1}
			cli.Publish(MsgData{
				Exchange: exchange,
				Queue:    queue,
				Content:  &v,
			})
		}
	}()
}

type Receiver interface {
	Group() sync.WaitGroup
	Channel() *amqp.Channel
	ExchangeName() string
	QueueName() string
	OnError(err error)
	OnReceive(b []byte) bool
}

// 监听对象
type PullReceiver struct {
	group    sync.WaitGroup
	channel  *amqp.Channel
	Exchange string
	Queue    string
	LisData  LisData
	Callback func(msg MsgData) (MsgData, error)
}

func (self *PullReceiver) Group() sync.WaitGroup {
	return self.group
}

func (self *PullReceiver) Channel() *amqp.Channel {
	return self.channel
}

func (self *PullReceiver) ExchangeName() string {
	return self.Exchange
}

func (self *PullReceiver) QueueName() string {
	return self.Queue
}

func (self *PullReceiver) OnError(err error) {
	log.Error(err.Error())
}

func (self *PullReceiver) OnReceive(b []byte) bool {
	if b == nil || len(b) == 0 || string(b) == "{}" {
		return true
	}
	log.Debug("消费数据日志", log.String("data", string(b)))
	message := MsgData{}
	if err := json.Unmarshal(b, &message); err != nil {
		log.Error("MQ消费数据转换JSON失败", log.String("exchange", self.Exchange), log.String("queue", self.Queue), log.String("data", string(b)))
	} else if message.Content == nil {
		log.Error("MQ消费数据Content为空", log.String("exchange", self.Exchange), log.String("queue", self.Queue), log.Any("data", message))
	} else if call, err := self.Callback(message); err != nil {
		log.Error("MQ消费数据处理异常", log.String("exchange", self.Exchange), log.String("queue", self.Queue), log.Any("data", call), log.AddError(err))
		if self.LisData.IsNack {
			return false
		}
	}
	return true
}
