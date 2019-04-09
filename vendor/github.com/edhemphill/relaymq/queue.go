package relaymq

import (
    "sync"
    "time"
    "strconv"
    "crypto/md5"
    "encoding/binary"
    "errors"
    "github.com/streadway/amqp"
    "crypto/rand"
    "fmt"
)

const ReconnectBackoffLimit = 32
const ResendBackoffLimit = 32

func randomString() string {
    randomBytes := make([]byte, 16)
    rand.Read(randomBytes)
    
    high := binary.BigEndian.Uint64(randomBytes[:8])
    low := binary.BigEndian.Uint64(randomBytes[8:])
    
    return fmt.Sprintf("%05x%05x", high, low)
}

type Message struct {
    Body string
    ID string
    DeliveryTag uint64
}

type AMQPConfig struct {
    Channels int
    Host string
    Port int
    Username string
    Password string
    PrefetchLimit int
}

type AMQPQueueManager struct {
    config AMQPConfig
    channels []*AMQPChannel
    prefetchLimit int
    connection *amqp.Connection
    stop chan bool
    stopped bool
}

func NewAMQPQueueManager(config *AMQPConfig) *AMQPQueueManager {
    newQueueManager := &AMQPQueueManager{
        config: *config,
        channels: make([]*AMQPChannel, config.Channels),
        stop: make(chan bool),
    }

    for i, _ := range newQueueManager.channels {
        newQueueManager.channels[i] = NewAMQPChannel(config.PrefetchLimit)
    }

    return newQueueManager
}

func (queueManager *AMQPQueueManager) Start() {
    uri := amqp.URI{
        Scheme: "amqp",
        Host: queueManager.config.Host,
        Port: queueManager.config.Port,
        Username: queueManager.config.Username,
        Password: queueManager.config.Password,
    }

    go queueManager.connect(uri.String())
}

func (queueManager *AMQPQueueManager) connect(uri string) {
    backoffTime := 1

    for {
        connection, err := amqp.Dial(uri)

        if err == nil {
            log.Infof("Connected to AMQP broker %s", uri)

            queueManager.connection = connection
            backoffTime = 1

            for i, channel := range queueManager.channels {
                go queueManager.startChannel(i, channel)
            }

            var closeError error
            connectionClose := connection.NotifyClose(make(chan *amqp.Error, 1))

            select {
                case closeError = <-connectionClose:
                case <-queueManager.stop:
                    log.Infof("Queue manager has been stopped. Shutting down...")
                    queueManager.disconnect()
                    return
            }

            for _, channel := range queueManager.channels {
                queueManager.stopChannel(channel)
            }

            log.Warningf("Connection with AMQP broker %s closed: %s. Reconnecting in %d seconds...", uri, closeError.Error(), backoffTime)
        } else {
            log.Warningf("Unable to establish connection with AMQP broker %s: %s. Reconnecting in %d seconds...", uri, err.Error(), backoffTime)
        }

        select {
            case <-time.After(time.Duration(backoffTime) * time.Second):
            case <-queueManager.stop:
                log.Infof("Queue manager has been stopped. Shutting down...")
                queueManager.disconnect()
                return
        }

        if backoffTime < ReconnectBackoffLimit {
            backoffTime *= 2
        }
    }
}

func (queueManager *AMQPQueueManager) disconnect() {
    for _, channel := range queueManager.channels {
        queueManager.stopChannel(channel)
    }

    if queueManager.connection != nil {
        queueManager.connection.Close()
    }
}

func (queueManager *AMQPQueueManager) startChannel(channelIndex int, channel *AMQPChannel) {
    backoffTime := 1

    for {
        amqpChannel, err := queueManager.connection.Channel()
        var stop chan error

        if err != nil {
            log.Warningf("Unable to establish AMQP channel %d: %s. Retrying in %d seconds...", channelIndex, err.Error(), backoffTime)
        } else {
            log.Infof("Established new channel %d", channelIndex)

            backoffTime = 1

            stop = channel.start(amqpChannel)

            select {
                case err := <-stop:
                    if err != nil {
                        log.Warningf("Channel %d has closed: %s. Reestablishing channel in %d seconds...", channelIndex, err.Error(), backoffTime)
                        break
                    }

                    log.Infof("Channel %d has been closed normally. Channel will not be reestablished", channelIndex)
                    return
            }
        }

        select {
            case <-time.After(time.Duration(backoffTime) * time.Second):
            case <-stop:
                log.Infof("Channel %d has been closed normally. Channel will not be reestablished", channelIndex)
                return
        }

        if backoffTime < ReconnectBackoffLimit {
            backoffTime *= 2
        }
    }
}

func (queueManager *AMQPQueueManager) stopChannel(channel *AMQPChannel) {
    channel.stop(nil)
}

func (queueManager *AMQPQueueManager) Stop() {
    if queueManager.stopped {
        return
    }

    queueManager.stop <- true
    queueManager.stopped = true
}

func (queueManager *AMQPQueueManager) Publish(queue string, message string) error {
    channel := queueManager.channel(queue)

    return channel.publish(queue, message)
}

func (queueManager *AMQPQueueManager) Subscribe(queue string) <-chan Message {
    channel := queueManager.channel(queue)

    return channel.subscribe(queue)
}

func (queueManager *AMQPQueueManager) Unsubscribe(queue string) {
    channel := queueManager.channel(queue)

    channel.unsubscribe(queue)
}

func (queueManager *AMQPQueueManager) Purge(queue string) error {
    channel := queueManager.channel(queue)

    return channel.purge(queue)
}

func (queueManager *AMQPQueueManager) Ack(queue string, messageID uint64) error {
    channel := queueManager.channel(queue)

    return channel.ack(queue, messageID)
}

func (queueManager *AMQPQueueManager) QueueSize(queue string) uint64 {
    channel := queueManager.channel(queue)

    return channel.queueSize(queue)
}

func (queueManager *AMQPQueueManager) channel(queueName string) *AMQPChannel {
    sum := md5.Sum([]byte(queueName))
    sumUint := binary.BigEndian.Uint64(sum[:8])

    return queueManager.channels[sumUint % uint64(len(queueManager.channels))]
}

type AMQPChannel struct {
    queues map[string]chan Message
    cancelNotifiers map[string]chan bool
    messages chan Message
    stopChan chan error
    channel *amqp.Channel
    publishAcks map[string]chan bool
    unackedMessages map[string]map[uint64]bool
    publishNextID uint64
    prefetchLimit int
    m sync.Mutex
}

func NewAMQPChannel(prefetchLimit int) *AMQPChannel {
    queue := &AMQPChannel{
        queues: make(map[string]chan Message),
        cancelNotifiers: make(map[string]chan bool),
        messages: make(chan Message),
        stopChan: make(chan error, 1),
        publishAcks: make(map[string]chan bool),
        prefetchLimit: prefetchLimit,
    }

    return queue
}

func (channel *AMQPChannel) subscribe(queue string) <-chan Message {
    channel.m.Lock()
    defer channel.m.Unlock()

    log.Debugf("Subscribing to queue %s", queue)

    // remove old subscriber from this queue if they exist
    if ch, ok := channel.queues[queue]; ok {
        log.Debugf("Already subscribed to queue %s. The existing message channel will be closed.", queue)
        
        close(ch)
        delete(channel.queues, queue)

        err := channel.removeQueue(queue)

        if err != nil {
            log.Warningf("removeQueue(%s) failed: %v", queue, err.Error())
        }
    }

    channel.queues[queue] = make(chan Message)

    err := channel.addQueue(queue)

    if err != nil {
        log.Warningf("addQueue(%s) failed: %v", queue, err.Error())
    }

    log.Debugf("Subscribed to queue %s", queue)

    return channel.queues[queue]
}

func (channel *AMQPChannel) unsubscribe(queue string) {
    channel.m.Lock()
    defer channel.m.Unlock()

    log.Debugf("Unsubscribing from queue %s", queue)

    err := channel.removeQueue(queue)

    if err != nil {
        log.Warningf("removeQueue(%s) failed: %v", queue, err.Error())
    }

    if ch, ok := channel.queues[queue]; ok {
        close(ch)
        delete(channel.queues, queue)
    }

    log.Debugf("Unsubscribed from queue %s", queue)
}

func (channel *AMQPChannel) purge(queue string) error {
    channel.m.Lock()
    defer channel.m.Unlock()

    if channel.channel == nil {
        return EEmpty
    }

    log.Debugf("Purging queue %s", queue)

    err := channel.removeQueue(queue)

    if err != nil {
        log.Errorf("removeQueue(%s) failed: %v", queue, err.Error())

        return err
    }

    if ch, ok := channel.queues[queue]; ok {
        close(ch)
        delete(channel.queues, queue)
    }

    queueName := channel.queueName(queue)
    _, err = channel.channel.QueueDelete(queueName, false, false, false)

    if err != nil {
        log.Errorf("Unable to delete queue %s: %s", queueName, err.Error())

        return err
    }

    log.Debugf("Deleted queue %s", queueName)

    return nil
}

func (channel *AMQPChannel) declare(queue string) error {
    log.Debugf("Declaring queue %s", queue)

    if channel.channel == nil {
        return EEmpty
    }

    queueName := channel.queueName(queue)
    _, err := channel.channel.QueueDeclare(queueName, true, false, false, false, nil)

    if err != nil {
        log.Errorf("Unable to declare queue %s: %v", queueName, err.Error())

        return err
    }

    log.Debugf("Declared queue %s", queue)

    return nil
}

func (channel *AMQPChannel) publish(queue string, message string) error {
    if channel.channel == nil {
        return errors.New("Channel not initialized")
    }

    err := channel.declare(queue)

    if err != nil {
        log.Errorf("Could not declare queue %s to publish message %s: %s", queue, message, err.Error())

        return err
    }
    
    channel.m.Lock()

    messageBody := []byte(message)
    queueName := channel.queueName(queue)
    messageID := channel.messageIDToString(channel.publishNextID)

    log.Debugf("Publishing message %s to queue %s: %v", messageID, queueName, message)

    msg := amqp.Publishing{
        DeliveryMode: amqp.Persistent,
        Timestamp: time.Now(),
        ContentType: "text/plain",
        Body: messageBody,
        MessageId: randomString(),
    }

    ackChan := make(chan bool, 1)
    channel.publishAcks[messageID] = ackChan

    err = channel.channel.Publish("", queueName, false, false, msg)

    if err != nil {
        log.Errorf("Unable to publish message %s to queue %s: %s", messageID, queueName, err.Error())
        delete(channel.publishAcks, messageID)
        channel.m.Unlock()

        return err
    }

    channel.publishNextID++
    channel.m.Unlock()

    result := <-ackChan

    channel.m.Lock()
    delete(channel.publishAcks, messageID)
    channel.m.Unlock()

    if !result {
        log.Errorf("Nack received for message %s to queue %s: %s", messageID, queue, message)

        return EEmpty
    }

    log.Debugf("Published message %s to queue %s: %s", messageID, queue, message)

    return nil
}

func (channel *AMQPChannel) ack(queue string, messageID uint64) error {
    if channel.channel == nil {
        return EEmpty
    }

    channel.m.Lock()
    if unackedMessages, ok := channel.unackedMessages[queue]; ok {
        delete(unackedMessages, messageID)
    }
    channel.m.Unlock()

    return channel.channel.Ack(messageID, false)
}

func (channel *AMQPChannel) messageIDToString(id uint64) string {
    return strconv.FormatUint(id, 10)
}

func (channel *AMQPChannel) consume() error {
    channel.m.Lock()
    defer channel.m.Unlock()

    for queue, _ := range channel.queues {
        err := channel.addQueue(queue)

        if err != nil {
            return err
        }
    }

    return nil
}

func (channel *AMQPChannel) addQueue(queue string) error {
    if channel.channel == nil {
        return EEmpty
    }

    err := channel.declare(queue)

    if err != nil {
        log.Errorf("Unable to declare queue %s in addQueue(%s): %s", queue, queue, err.Error())

        return err
    }

    queueName := channel.queueName(queue)
    deliveryChan, err := channel.channel.Consume(queueName, queue, false, false, false, false, nil)

    if err != nil {
        return err
    }

    go func() {
        channel.m.Lock()
        unackedMessages := make(map[uint64]bool)
        channel.unackedMessages[queueName] = unackedMessages
        channel.cancelNotifiers[queue] = make(chan bool, 1)
        channel.m.Unlock()

        defer func() {
            log.Debugf("Cancel delivery channel read for queue %s", queue)
            
            channel.m.Lock()
            for deliveryTag, _ := range unackedMessages {
                channel.channel.Nack(deliveryTag, false, true)
            }

            delete(channel.unackedMessages, queueName)
            channel.m.Unlock()
        }()

        for delivery := range deliveryChan {
            consumeChan := channel.queues[queue]
            message, err := channel.deliveryToMessage(delivery)

            if err != nil {
                log.Warningf("Received an invalid message %s from queue %s: %v. Discarding...", string(delivery.Body), queue, err.Error())
                delivery.Ack(false) // drop this message. it is not valid
                continue
            }

            channel.m.Lock()
            unackedMessages[message.DeliveryTag] = true
            cancelNotifier, ok := channel.cancelNotifiers[queue]
            channel.m.Unlock()

            if !ok {
                continue
            }

            select {
                case <-cancelNotifier:
                    continue
                case consumeChan <- *message:
            }
        }
    }()

    return nil
}

func (queue *AMQPChannel) deliveryToMessage(delivery amqp.Delivery) (*Message, error) {
    message := &Message{
        Body: string(delivery.Body),
        ID: delivery.MessageId,        
        DeliveryTag: delivery.DeliveryTag,
    }

    return message, nil
}

func (channel *AMQPChannel) queueName(queue string) string {
    return queue
}

func (channel *AMQPChannel) removeQueue(queue string) error {
    if channel.channel == nil {
        return EEmpty
    }

    err := channel.channel.Cancel(queue, false)

    cancelNotifier, ok := channel.cancelNotifiers[queue]

    if ok {
        cancelNotifier <- true
        delete(channel.cancelNotifiers, queue)
    }

    return err
}

func (channel *AMQPChannel) start(amqpChannel *amqp.Channel) chan error {
    channel.m.Lock()
    channel.publishNextID = 1
    channel.m.Unlock()

    channel.channel = amqpChannel
    channelClose := amqpChannel.NotifyClose(make(chan *amqp.Error))
    channelCancel := amqpChannel.NotifyCancel(make(chan string))
    channelAck, channelNack  := amqpChannel.NotifyConfirm(make(chan uint64), make(chan uint64))

    channel.unackedMessages = make(map[string]map[uint64]bool)

    go func() {
        for topic := range channelCancel {
            channel.m.Lock()

            if notifier, ok := channel.cancelNotifiers[topic]; ok {
                close(notifier)
                delete(channel.cancelNotifiers, topic)
            }

            channel.m.Unlock()
        }
    }()

    go func() {
        for ackMessageID := range channelAck {
            channel.m.Lock()
            ackChan, ok := channel.publishAcks[channel.messageIDToString(ackMessageID)]

            if !ok {
                channel.m.Unlock()
                continue
            }

            ackChan <- true
            channel.m.Unlock()
        }
    }()

    go func() {
        for nackMessageID := range channelNack {
            channel.m.Lock()
            ackChan, ok := channel.publishAcks[channel.messageIDToString(nackMessageID)]

            if !ok {
                channel.m.Unlock()
                continue
            }

            ackChan <- false
            channel.m.Unlock()
        }
    }()

    go func() {
        err := <-channelClose

        channel.m.Lock()

        for topic, notifier := range channel.cancelNotifiers {
            close(notifier)
            delete(channel.cancelNotifiers, topic)
        }

        for ackMessageID, ackChan := range channel.publishAcks {
            ackChan <- false
            delete(channel.publishAcks, ackMessageID)
        }

        channel.m.Unlock()

        if err != nil {
            channel.stop(err)
        }
    }()

    err := amqpChannel.Confirm(false)

    if err != nil {
        log.Errorf("Unable to put channel into confirm mode. Reason: %v", err.Error())

        channel.stopChan <- err

        return channel.stopChan
    }

    // this ensures that each consumer gets at most one unacknowledged message. This can be adjusted
    // for faster message processing
    err = amqpChannel.Qos(channel.prefetchLimit, 0, false)

    if err != nil {
        log.Errorf("Unable to set Qos for channel. Reason: %v", err.Error())

        channel.stopChan <- err

        return channel.stopChan
    }

    err = channel.consume()

    if err != nil {
        log.Errorf("Unable to consume from channel. Reason: %v", err.Error())

        channel.stopChan <- err
    }

    return channel.stopChan
}

func (channel *AMQPChannel) stop(err error) {
    if channel.channel != nil {
        channel.channel.Close()
    }

    channel.stopChan <- err
}

func (channel *AMQPChannel) queueSize(queue string) uint64 {
    if channel.channel == nil {
        return 0
    }

    queueName := channel.queueName(queue)
    q, err := channel.channel.QueueInspect(queueName)

    if err != nil {
        log.Errorf("Unable to inspect queue %s: %v", queueName, err.Error())

        return 0
    }

    return uint64(q.Messages)
}
