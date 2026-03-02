package cli

import (
	"context"
	"strings"

	"github.com/nanobotgo/agent"
	"github.com/nanobotgo/bus"
	"github.com/nanobotgo/utils"
)

type AgentDispatcher struct {
	agentLoops   map[string]*agent.AgentLoop
	defaultAgent *agent.AgentLoop
	bus          *bus.MessageBus
}

func NewAgentDispatcher(agentLoops map[string]*agent.AgentLoop, defaultAgent *agent.AgentLoop, bus *bus.MessageBus) *AgentDispatcher {
	return &AgentDispatcher{
		agentLoops:   agentLoops,
		defaultAgent: defaultAgent,
		bus:          bus,
	}
}

func (d *AgentDispatcher) StartConsuming(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := d.bus.ConsumeInbound(ctx)
				if err != nil {
					return
				}
				if msg != nil {
					go func() {
						_, err := d.ProcessMessage(ctx, msg)
						if err != nil {
							utils.Log.Errorf("Process message error: %v", err)
						}
					}()
				}
			}
		}
	}()
}

func (d *AgentDispatcher) GetAgentForMessage(content string) *agent.AgentLoop {
	agentName := d.ParseAgentMention(content)
	if agentName == "default" {
		return d.defaultAgent
	}

	if agent, ok := d.agentLoops[agentName]; ok {
		return agent
	}

	utils.Log.Warnf("Agent [%s] not found, using default", agentName)
	return d.defaultAgent
}

func (d *AgentDispatcher) ParseAgentMention(content string) string {
	content = strings.TrimSpace(content)

	for name := range d.agentLoops {
		if name == "default" {
			continue
		}
		mention := "@" + name
		if strings.Contains(content, mention) {
			return name
		}
	}

	return "default"
}

func (d *AgentDispatcher) ProcessMessage(ctx context.Context, msg *bus.InboundMessage) (string, error) {
	agentName := d.ParseAgentMention(msg.Content)

	dispatched := d.bus.DispatchToAgent(ctx, agentName, msg)
	if !dispatched {
		utils.Log.Warnf("Failed to dispatch message to agent [%s], falling back to default", agentName)
		agentLoop := d.defaultAgent
		return agentLoop.ProcessDirect(ctx, msg.Content, msg.ChatID, msg.Channel, msg.ChatID, msg.Media)
	}
	return "", nil
}

func (d *AgentDispatcher) StopAll() {
	for _, agentLoop := range d.agentLoops {
		agentLoop.Stop()
	}
}

func (d *AgentDispatcher) GetAgentLoops() map[string]*agent.AgentLoop {
	return d.agentLoops
}
