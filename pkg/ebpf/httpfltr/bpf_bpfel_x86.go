// Code generated by bpf2go; DO NOT EDIT.
//go:build 386 || amd64
// +build 386 amd64

package httpfltr

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"

	"github.com/cilium/ebpf"
)

type bpfHttpConnectionInfoT struct {
	S_addr [16]uint8
	D_addr [16]uint8
	S_port uint16
	D_port uint16
	Flags  uint32
}

type bpfHttpConnectionMetadataT struct {
	Id    uint64
	Flags uint8
	_     [7]byte
}

type bpfSockArgsT struct {
	Addr       uint64
	AcceptTime uint64
}

// loadBpf returns the embedded CollectionSpec for bpf.
func loadBpf() (*ebpf.CollectionSpec, error) {
	reader := bytes.NewReader(_BpfBytes)
	spec, err := ebpf.LoadCollectionSpecFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("can't load bpf: %w", err)
	}

	return spec, err
}

// loadBpfObjects loads bpf and converts it into a struct.
//
// The following types are suitable as obj argument:
//
//	*bpfObjects
//	*bpfPrograms
//	*bpfMaps
//
// See ebpf.CollectionSpec.LoadAndAssign documentation for details.
func loadBpfObjects(obj interface{}, opts *ebpf.CollectionOptions) error {
	spec, err := loadBpf()
	if err != nil {
		return err
	}

	return spec.LoadAndAssign(obj, opts)
}

// bpfSpecs contains maps and programs before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type bpfSpecs struct {
	bpfProgramSpecs
	bpfMapSpecs
}

// bpfSpecs contains programs before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type bpfProgramSpecs struct {
	KprobeSysExit       *ebpf.ProgramSpec `ebpf:"kprobe_sys_exit"`
	KprobeTcpConnect    *ebpf.ProgramSpec `ebpf:"kprobe_tcp_connect"`
	KretprobeSockAlloc  *ebpf.ProgramSpec `ebpf:"kretprobe_sock_alloc"`
	KretprobeSysAccept4 *ebpf.ProgramSpec `ebpf:"kretprobe_sys_accept4"`
	KretprobeSysConnect *ebpf.ProgramSpec `ebpf:"kretprobe_sys_connect"`
	SocketHttpFilter    *ebpf.ProgramSpec `ebpf:"socket__http_filter"`
}

// bpfMapSpecs contains maps before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type bpfMapSpecs struct {
	ActiveAcceptArgs    *ebpf.MapSpec `ebpf:"active_accept_args"`
	ActiveConnectArgs   *ebpf.MapSpec `ebpf:"active_connect_args"`
	DeadPids            *ebpf.MapSpec `ebpf:"dead_pids"`
	Events              *ebpf.MapSpec `ebpf:"events"`
	FilteredConnections *ebpf.MapSpec `ebpf:"filtered_connections"`
	HttpTcpSeq          *ebpf.MapSpec `ebpf:"http_tcp_seq"`
}

// bpfObjects contains all objects after they have been loaded into the kernel.
//
// It can be passed to loadBpfObjects or ebpf.CollectionSpec.LoadAndAssign.
type bpfObjects struct {
	bpfPrograms
	bpfMaps
}

func (o *bpfObjects) Close() error {
	return _BpfClose(
		&o.bpfPrograms,
		&o.bpfMaps,
	)
}

// bpfMaps contains all maps after they have been loaded into the kernel.
//
// It can be passed to loadBpfObjects or ebpf.CollectionSpec.LoadAndAssign.
type bpfMaps struct {
	ActiveAcceptArgs    *ebpf.Map `ebpf:"active_accept_args"`
	ActiveConnectArgs   *ebpf.Map `ebpf:"active_connect_args"`
	DeadPids            *ebpf.Map `ebpf:"dead_pids"`
	Events              *ebpf.Map `ebpf:"events"`
	FilteredConnections *ebpf.Map `ebpf:"filtered_connections"`
	HttpTcpSeq          *ebpf.Map `ebpf:"http_tcp_seq"`
}

func (m *bpfMaps) Close() error {
	return _BpfClose(
		m.ActiveAcceptArgs,
		m.ActiveConnectArgs,
		m.DeadPids,
		m.Events,
		m.FilteredConnections,
		m.HttpTcpSeq,
	)
}

// bpfPrograms contains all programs after they have been loaded into the kernel.
//
// It can be passed to loadBpfObjects or ebpf.CollectionSpec.LoadAndAssign.
type bpfPrograms struct {
	KprobeSysExit       *ebpf.Program `ebpf:"kprobe_sys_exit"`
	KprobeTcpConnect    *ebpf.Program `ebpf:"kprobe_tcp_connect"`
	KretprobeSockAlloc  *ebpf.Program `ebpf:"kretprobe_sock_alloc"`
	KretprobeSysAccept4 *ebpf.Program `ebpf:"kretprobe_sys_accept4"`
	KretprobeSysConnect *ebpf.Program `ebpf:"kretprobe_sys_connect"`
	SocketHttpFilter    *ebpf.Program `ebpf:"socket__http_filter"`
}

func (p *bpfPrograms) Close() error {
	return _BpfClose(
		p.KprobeSysExit,
		p.KprobeTcpConnect,
		p.KretprobeSockAlloc,
		p.KretprobeSysAccept4,
		p.KretprobeSysConnect,
		p.SocketHttpFilter,
	)
}

func _BpfClose(closers ...io.Closer) error {
	for _, closer := range closers {
		if err := closer.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Do not access this directly.
//
//go:embed bpf_bpfel_x86.o
var _BpfBytes []byte
