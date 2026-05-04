package kube

// DNSSignals holds the status of cluster DNS (CoreDNS)
type DNSSignals struct {
	ServiceFound   bool
	EndpointsReady int
	TotalEndpoints int
}
