// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package extensions

import (
	"github.com/google/go-tdx-guest/proto/tdx"
	"google.golang.org/protobuf/proto"
)

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative extensions.proto

func Unmarshal(data []byte) (*tdx.QuoteV4, []*Extension, error) {
	var quote tdx.QuoteV4
	if err := proto.Unmarshal(data, &quote); err != nil {
		return nil, nil, err
	}

	var trailer Trailer
	if err := proto.Unmarshal(data, &trailer); err != nil {
		return nil, nil, err
	}

	return &quote, trailer.Extensions, nil
}

func Marshal(quote *tdx.QuoteV4, extensions ...*Extension) ([]byte, error) {
	quoteBytes, err := proto.Marshal(quote)
	if err != nil {
		return nil, err
	}
	trailer := &Trailer{Extensions: extensions}
	trailerBytes, err := proto.Marshal(trailer)
	if err != nil {
		return nil, err
	}
	return append(quoteBytes, trailerBytes...), nil
}
