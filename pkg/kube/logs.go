package kube

import (
    "bytes"
    "context"
    "io"
    "strings"

    corev1 "k8s.io/api/core/v1"
    "k8s.io/client-go/kubernetes"
)

// CollectLogs fetches recent logs from a container.
// Tries previous container first (crash logs),
// falls back to current container if no previous exists.
func CollectLogs(
    client *kubernetes.Clientset,
    podName, namespace, container string,
    tailLines int64,
) (string, error) {

    // Try previous container first —
    // crash logs are more useful than current logs
    logs, err := fetchLogs(
        client, podName, namespace, container,
        tailLines, true)
    if err == nil && strings.TrimSpace(logs) != "" {
        return logs, nil
    }

    // Fall back to current container
    return fetchLogs(
        client, podName, namespace, container,
        tailLines, false)
}

func fetchLogs(
    client *kubernetes.Clientset,
    podName, namespace, container string,
    tailLines int64,
    previous bool,
) (string, error) {

    ctx := context.Background()

    opts := &corev1.PodLogOptions{
        Container: container,
        TailLines: &tailLines,
        Previous:  previous,
    }

    req := client.CoreV1().Pods(namespace).
        GetLogs(podName, opts)

    stream, err := req.Stream(ctx)
    if err != nil {
        return "", err
    }
    defer stream.Close()

    var buf bytes.Buffer
    if _, err = io.Copy(&buf, stream); err != nil {
        return "", err
    }

    return buf.String(), nil
}
