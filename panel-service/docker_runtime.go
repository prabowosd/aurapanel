package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func containerRuntimeCommand() (string, error) {
	for _, candidate := range []string{"docker", "podman"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("docker or podman not found")
}

func runtimeDockerContainers() ([]DockerContainer, error) {
	command, err := containerRuntimeCommand()
	if err != nil {
		return nil, err
	}
	output, err := commandOutputTrimmed(command, "ps", "-a", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}\t{{.RunningFor}}")
	if err != nil && strings.TrimSpace(err.Error()) == "" {
		return []DockerContainer{}, nil
	}
	if err != nil {
		return nil, err
	}
	containers := []DockerContainer{}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 6 {
			continue
		}
		containers = append(containers, DockerContainer{
			ID:      fields[0],
			Name:    fields[1],
			Image:   fields[2],
			Status:  fields[3],
			Ports:   fields[4],
			Created: fields[5],
		})
	}
	return containers, nil
}

func runtimeDockerImages() ([]DockerImage, error) {
	command, err := containerRuntimeCommand()
	if err != nil {
		return nil, err
	}
	output, err := commandOutputTrimmed(command, "images", "--format", "{{.ID}}\t{{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedSince}}")
	if err != nil && strings.TrimSpace(err.Error()) == "" {
		return []DockerImage{}, nil
	}
	if err != nil {
		return nil, err
	}
	images := []DockerImage{}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 5 {
			continue
		}
		images = append(images, DockerImage{
			ID:         fields[0],
			Repository: fields[1],
			Tag:        fields[2],
			Size:       fields[3],
			Created:    fields[4],
		})
	}
	return images, nil
}

func createRuntimeDockerContainer(name, image string, ports []string, restartPolicy string, memoryLimit string, cpuLimit string, env []string, volumes []string) error {
	command, err := containerRuntimeCommand()
	if err != nil {
		return err
	}
	name = sanitizeName(name)
	if name == "" {
		return fmt.Errorf("container name is required")
	}
	if _, err := commandOutputTrimmed(command, "pull", image); err != nil {
		return err
	}
	args := []string{"run", "-d", "--name", name}
	if strings.TrimSpace(restartPolicy) != "" {
		args = append(args, "--restart", strings.TrimSpace(restartPolicy))
	}
	if strings.TrimSpace(memoryLimit) != "" {
		args = append(args, "-m", strings.TrimSpace(memoryLimit))
	}
	if strings.TrimSpace(cpuLimit) != "" {
		args = append(args, "--cpus", strings.TrimSpace(cpuLimit))
	}
	for _, envVar := range env {
		envVar = strings.TrimSpace(envVar)
		if envVar != "" {
			args = append(args, "-e", envVar)
		}
	}
	for _, volume := range volumes {
		volume = strings.TrimSpace(volume)
		if volume != "" {
			args = append(args, "-v", volume)
		}
	}
	for _, port := range ports {
		port = strings.TrimSpace(port)
		if port != "" {
			args = append(args, "-p", port)
		}
	}
	args = append(args, image)
	_, err = commandOutputTrimmed(command, args...)
	return err
}

func applyRuntimeDockerContainerAction(id, action string) error {
	command, err := containerRuntimeCommand()
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("container id is required")
	}
	switch action {
	case "start", "stop", "restart":
		_, err = commandOutputTrimmed(command, action, id)
	case "remove":
		_, err = commandOutputTrimmed(command, "rm", "-f", id)
	default:
		return fmt.Errorf("unsupported container action")
	}
	return err
}

func pullRuntimeDockerImage(image, tag string) error {
	command, err := containerRuntimeCommand()
	if err != nil {
		return err
	}
	image = strings.TrimSpace(image)
	tag = firstNonEmpty(strings.TrimSpace(tag), "latest")
	if image == "" {
		return fmt.Errorf("image is required")
	}
	_, err = commandOutputTrimmed(command, "pull", image+":"+tag)
	return err
}

func removeRuntimeDockerImage(id string) error {
	command, err := containerRuntimeCommand()
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("image id is required")
	}
	_, err = commandOutputTrimmed(command, "rmi", "-f", id)
	return err
}
