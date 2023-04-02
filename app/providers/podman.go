package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/acouvreur/sablier/app/instance"
	"github.com/containers/podman/v4/cmd/podman/registry"
	"github.com/containers/podman/v4/libpod/events"
	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/containers"
	"github.com/containers/podman/v4/pkg/domain/entities"
	log "github.com/sirupsen/logrus"
)

type PodmanProvider struct {
	Client          context.Context
	desiredReplicas int
}

func NewPodmanProvider() (*PodmanProvider, error) {
	conn, err := bindings.NewConnection(context.Background(), "unix://run/podman/podman.sock")
	if err != nil {
		log.Fatal(fmt.Errorf("%+v", "Could not create Podman binding"))
		return nil, err
	}

	return &PodmanProvider{
		Client:          conn,
		desiredReplicas: 1,
	}, nil
}

func (provider *PodmanProvider) GetGroups() (map[string][]string, error) {

	all := true
	containers, err := containers.List(provider.Client, &containers.ListOptions{
		All: &all,
		Filters: map[string][]string{
			"label": []string{fmt.Sprintf("%s=true", enableLabel)},
		},
	})

	if err != nil {
		return nil, err
	}

	groups := make(map[string][]string)
	for _, container := range containers {
		groupName := container.Labels[groupLabel]
		if len(groupName) == 0 {
			groupName = defaultGroupValue
		}
		group := groups[groupName]
		group = append(group, strings.TrimPrefix(container.Names[0], "/"))
		groups[groupName] = group
	}

	log.Debug(fmt.Sprintf("%v", groups))

	return groups, nil
}

func (provider *PodmanProvider) Start(name string) (instance.State, error) {
	err := containers.Start(provider.Client, name, &containers.StartOptions{})

	if err != nil {
		return instance.ErrorInstanceState(name, err, provider.desiredReplicas)
	}

	return instance.State{
		Name:            name,
		CurrentReplicas: 0,
		DesiredReplicas: provider.desiredReplicas,
		Status:          instance.NotReady,
	}, err
}

func (provider *PodmanProvider) Stop(name string) (instance.State, error) {
	err := containers.Stop(provider.Client, name, &containers.StopOptions{})

	if err != nil {
		return instance.ErrorInstanceState(name, err, provider.desiredReplicas)
	}

	return instance.State{
		Name:            name,
		CurrentReplicas: 0,
		DesiredReplicas: provider.desiredReplicas,
		Status:          instance.NotReady,
	}, nil
}

func (provider *PodmanProvider) GetState(name string) (instance.State, error) {
	spec, err := containers.Inspect(provider.Client, name, &containers.InspectOptions{})

	if err != nil {
		return instance.ErrorInstanceState(name, err, provider.desiredReplicas)
	}

	// "created", "running", "paused", "restarting", "removing", "exited", or "dead"
	switch spec.State.Status {
	case "created", "paused", "restarting", "removing":
		return instance.NotReadyInstanceState(name, 0, provider.desiredReplicas)
	case "running":
		if spec.State.Health.Status != "" {
			// // "starting", "healthy" or "unhealthy"
			if spec.State.Health.Status == "healthy" {
				return instance.ReadyInstanceState(name, provider.desiredReplicas)
			} else if spec.State.Health.Status == "unhealthy" {
				if len(spec.State.Health.Log) >= 1 {
					lastLog := spec.State.Health.Log[len(spec.State.Health.Log)-1]
					return instance.UnrecoverableInstanceState(name, fmt.Sprintf("container is unhealthy: %s (%d)", lastLog.Output, lastLog.ExitCode), provider.desiredReplicas)
				} else {
					return instance.UnrecoverableInstanceState(name, "container is unhealthy: no log available", provider.desiredReplicas)
				}
			} else {
				return instance.NotReadyInstanceState(name, 0, provider.desiredReplicas)
			}
		}
		return instance.ReadyInstanceState(name, provider.desiredReplicas)
	case "exited":
		if spec.State.ExitCode != 0 {
			return instance.UnrecoverableInstanceState(name, fmt.Sprintf("container exited with code \"%d\"", spec.State.ExitCode), provider.desiredReplicas)
		}
		return instance.NotReadyInstanceState(name, 0, provider.desiredReplicas)
	case "dead":
		return instance.UnrecoverableInstanceState(name, "container in \"dead\" state cannot be restarted", provider.desiredReplicas)
	default:
		return instance.UnrecoverableInstanceState(name, fmt.Sprintf("container status \"%s\" not handled", spec.State.Status), provider.desiredReplicas)
	}
}

func (provider *PodmanProvider) NotifyInstanceStopped(ctx context.Context, instance chan<- string) {
	eventChannel := make(chan *events.Event, 1)
	errChannel := make(chan error)

	registry.ContainerEngine().Events(context.Background(), entities.EventsOptions{
		EventChan: eventChannel,
		Filter: []string{
			"event=die",
		},
	})

	for {
		select {
		case event, ok := <-eventChannel:
			if !ok {
				// channel was closed we can exit
				return
			}
			instance <- strings.TrimPrefix(event.Attributes["name"], "/")

		case err := <-errChannel:
			// only exit in case of an error,
			// otherwise keep reading events until the event channel is closed
			if err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
