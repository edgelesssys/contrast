package main

import (
	"bytes"
	"io"
	"os"

	"github.com/edgelesssys/contrast/tools/igvm/defs"
)

func main() {
	f, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	if len(f) < 0x16 {
		panic("file too small")
	}

	var igvm IGVM
	if err := igvm.BinaryUnmarshal(f); err != nil {
		panic(err)
	}

	var idblock defs.IgvmVhsSnpIDBlock
	for i, vhs := range igvm.VariableHeaders {
		if vhs.Type == defs.IgvmVhtSnpIdBlock {
			if err := idblock.BinaryUnmarshal(vhs.Content); err != nil {
				panic(err)
			}
			idblockBytes, err := idblock.BinaryMarshal()
			if err != nil {
				panic(err)
			}
			igvm.VariableHeaders[i].Content = idblockBytes
			break
		}
	}

	igvmData, err := igvm.BinaryMarshal()
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, bytes.NewBuffer(igvmData))
}

type IGVM struct {
	Header          defs.IGVMFixedHeader
	VariableHeaders []defs.IGVMVHSVariableHeader
	FileData        []byte
}

func (igvm *IGVM) BinaryMarshal() ([]byte, error) {
	// Fixed header
	data, err := igvm.Header.BinaryMarshal()
	if err != nil {
		return nil, err
	}
	for _, vhs := range igvm.VariableHeaders {
		vhsData, err := vhs.BinaryMarshal()
		if err != nil {
			return nil, err
		}
		data = append(data, vhsData...)
	}
	data = append(data, igvm.FileData...)
	return data, nil
}

func (igvm *IGVM) BinaryUnmarshal(data []byte) error {
	// Fixed header
	if err := igvm.Header.BinaryUnmarshal(data[:24]); err != nil {
		return err
	}
	index := igvm.Header.VariableHeaderOffset
	for index < igvm.Header.VariableHeaderOffset+igvm.Header.VariableHeaderSize {
		var vhs defs.IGVMVHSVariableHeader
		if err := vhs.BinaryUnmarshal(data[index:]); err != nil {
			return err
		}
		igvm.VariableHeaders = append(igvm.VariableHeaders, vhs)
		index += 8 + vhs.Length + uint32(len(vhs.Padding))
	}
	igvm.FileData = data[index:]
	return nil
}
