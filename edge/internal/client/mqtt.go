package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

// MQTTClient wraps the autopaho ConnectionManager for edge agent operations.
type MQTTClient struct {
	cm        *autopaho.ConnectionManager
	brokerURL string
	clientID  string
	deviceID  string

	// Connection state
	mu          sync.RWMutex
	connected   bool
	onConnUp    func()
	onConnDown  func(error)

	// Message handler for commands
	cmdHandler func(topic string, payload []byte)
}

// NewMQTTClient creates a new MQTTClient for the edge agent.
func NewMQTTClient(brokerURL, clientID, deviceID string) *MQTTClient {
	return &MQTTClient{
		brokerURL: brokerURL,
		clientID:  clientID,
		deviceID:  deviceID,
	}
}

// SetOnConnectionUp sets the callback for when connection is established.
func (m *MQTTClient) SetOnConnectionUp(callback func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onConnUp = callback
}

// SetOnConnectionDown sets the callback for when connection is lost.
func (m *MQTTClient) SetOnConnectionDown(callback func(error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onConnDown = callback
}

// IsConnected returns the current connection state.
func (m *MQTTClient) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// Connect establishes connection to the MQTT broker with auto-reconnect.
func (m *MQTTClient) Connect(ctx context.Context) error {
	serverURL, err := url.Parse(m.brokerURL)
	if err != nil {
		return fmt.Errorf("invalid broker URL: %w", err)
	}

	cliCfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{serverURL},
		KeepAlive:                     20,
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval:         60,
		OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			log.Printf("INFO: MQTT connection established to %s", m.brokerURL)
			m.mu.Lock()
			m.connected = true
			callback := m.onConnUp
			m.mu.Unlock()

			if callback != nil {
				callback()
			}
		},
		OnConnectError: func(err error) {
			log.Printf("WARN: MQTT connection error: %v", err)
			m.mu.Lock()
			wasConnected := m.connected
			m.connected = false
			callback := m.onConnDown
			m.mu.Unlock()

			if wasConnected && callback != nil {
				callback(err)
			}
		},
		ClientConfig: paho.ClientConfig{
			ClientID: m.clientID,
			OnClientError: func(err error) {
				log.Printf("ERROR: MQTT client error: %v", err)
			},
			OnServerDisconnect: func(d *paho.Disconnect) {
				reasonStr := ""
				if d.Properties != nil {
					reasonStr = d.Properties.ReasonString
				}
				log.Printf("WARN: MQTT server disconnect: code=%d reason=%s", d.ReasonCode, reasonStr)
				m.mu.Lock()
				m.connected = false
				callback := m.onConnDown
				m.mu.Unlock()

				if callback != nil {
					callback(fmt.Errorf("server disconnect: code=%d", d.ReasonCode))
				}
			},
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(pr paho.PublishReceived) (bool, error) {
					m.handleMessage(pr.Packet)
					return true, nil
				},
			},
		},
	}

	cm, err := autopaho.NewConnection(ctx, cliCfg)
	if err != nil {
		return fmt.Errorf("failed to create MQTT connection: %w", err)
	}

	m.cm = cm

	// Wait for initial connection with timeout
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := cm.AwaitConnection(connectCtx); err != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", err)
	}

	return nil
}

// handleMessage routes incoming messages to the appropriate handler.
func (m *MQTTClient) handleMessage(p *paho.Publish) {
	m.mu.RLock()
	handler := m.cmdHandler
	m.mu.RUnlock()

	// Check if this is a command message for this device
	cmdPrefix := fmt.Sprintf("orion/edge/%s/cmd/", m.deviceID)
	if strings.HasPrefix(p.Topic, cmdPrefix) && handler != nil {
		handler(p.Topic, p.Payload)
	}
}

// Close disconnects from the MQTT broker.
func (m *MQTTClient) Close(ctx context.Context) error {
	if m.cm == nil {
		return nil
	}

	log.Printf("INFO: Disconnecting from MQTT broker...")

	disconnectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return m.cm.Disconnect(disconnectCtx)
}

// PublishHealth publishes a health message with QoS 1.
func (m *MQTTClient) PublishHealth(ctx context.Context, health map[string]interface{}) error {
	if m.cm == nil {
		return fmt.Errorf("MQTT client not connected")
	}

	topic := fmt.Sprintf("orion/edge/%s/health", m.deviceID)

	payload, err := json.Marshal(health)
	if err != nil {
		return fmt.Errorf("failed to marshal health: %w", err)
	}

	_, err = m.cm.Publish(ctx, &paho.Publish{
		Topic:   topic,
		QoS:     1, // At least once delivery for health messages
		Payload: payload,
	})

	if err != nil {
		return fmt.Errorf("failed to publish health: %w", err)
	}

	return nil
}

// SubscribeCommands subscribes to command topics and calls handler for each message.
func (m *MQTTClient) SubscribeCommands(ctx context.Context, handler func(topic string, payload []byte)) error {
	if m.cm == nil {
		return fmt.Errorf("MQTT client not connected")
	}

	topic := fmt.Sprintf("orion/edge/%s/cmd/#", m.deviceID)

	// Register the handler
	m.mu.Lock()
	m.cmdHandler = handler
	m.mu.Unlock()

	// Subscribe with QoS 1
	_, err := m.cm.Subscribe(ctx, &paho.Subscribe{
		Subscriptions: []paho.SubscribeOptions{
			{
				Topic: topic,
				QoS:   1,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe to commands: %w", err)
	}

	log.Printf("INFO: Subscribed to MQTT topic %s", topic)
	return nil
}
