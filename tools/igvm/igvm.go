package igvm

type IGVM struct {
	Header          FixedHeader
	VariableHeaders []VariableHeader
	FileData        []byte
}

// BinaryMarshal marshals the IGVM struct into a byte slice.
func (i *IGVM) BinaryMarshal() ([]byte, error) {
	data, err := i.Header.BinaryMarshal()
	if err != nil {
		return nil, err
	}
	for _, vhs := range i.VariableHeaders {
		vhsData, err := vhs.BinaryMarshal()
		if err != nil {
			return nil, err
		}
		data = append(data, vhsData...)
	}
	data = append(data, i.FileData...)
	return data, nil
}

// BinaryUnmarshal unmarshals the byte slice into the IGVM struct.
func (i *IGVM) BinaryUnmarshal(data []byte) error {
	if err := i.Header.BinaryUnmarshal(data[:24]); err != nil {
		return err
	}
	index := i.Header.VariableHeaderOffset
	for index < i.Header.VariableHeaderOffset+i.Header.VariableHeaderSize {
		var vhs VariableHeader
		if err := vhs.BinaryUnmarshal(data[index:]); err != nil {
			return err
		}
		i.VariableHeaders = append(i.VariableHeaders, vhs)
		index += 8 + vhs.Length + uint32(len(vhs.Padding))
	}
	i.FileData = data[index:]
	return nil
}
