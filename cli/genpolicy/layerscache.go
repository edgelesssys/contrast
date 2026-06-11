// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package genpolicy

import (
	"encoding/json"
	"fmt"
)

// NewLayersCache creates a new LayersCache from a slice of ImageLayersCacheEntry.
func NewLayersCache(entries []ImageLayersCacheEntry) (*LayersCache, error) {
	cache := &LayersCache{
		Index:  make(map[string]ImageLayerIndex),
		Layers: make(map[string]ImageLayer),
	}

	for _, entry := range entries {
		switch {
		case entry.Layer != nil:
			cache.Layers[entry.Layer.DiffID] = *entry.Layer

		case entry.ImageIndex != nil:
			cache.Index[entry.ImageIndex.ImageRef] = *entry.ImageIndex

		default:
			return nil, fmt.Errorf("invalid cache entry")
		}
	}

	return cache, nil
}

// LayersCache stores the layers DiffIDs and layer sizes for image references.
type LayersCache struct {
	Index  map[string]ImageLayerIndex
	Layers map[string]ImageLayer
}

// ImageLayer represents a single layer of an image, identified by its DiffID.
type ImageLayer struct {
	DiffID           string `json:"diff_id"`
	Passwd           string `json:"passwd"`
	Group            string `json:"group"`
	UncompressedSize uint64 `json:"uncompressed_size"`
}

// ImageLayerIndex maps an image reference to its layers.
type ImageLayerIndex struct {
	ImageRef string                 `json:"image_ref"`
	Layers   []ImageLayerIndexEntry `json:"layers"`
}

// ImageLayerIndexEntry represents a single layer entry in the image index, containing the DiffID and compressed size.
type ImageLayerIndexEntry struct {
	DiffID         string `json:"diff_id"`
	CompressedSize uint64 `json:"compressed_size"`
}

// ImageLayersCacheEntry represents a single entry in the layers cache, which can be either an ImageLayer or an ImageLayerIndex.
type ImageLayersCacheEntry struct {
	Layer      *ImageLayer
	ImageIndex *ImageLayerIndex
}

// UnmarshalJSON sets either the Layer or ImageIndex field based on the JSON data.
func (e *ImageLayersCacheEntry) UnmarshalJSON(data []byte) error {
	var layer ImageLayer
	if err := json.Unmarshal(data, &layer); err == nil &&
		layer.DiffID != "" {
		e.Layer = &layer
		return nil
	}

	var index ImageLayerIndex
	if err := json.Unmarshal(data, &index); err == nil &&
		index.ImageRef != "" {
		e.ImageIndex = &index
		return nil
	}

	return fmt.Errorf("unknown ImageLayersCacheEntry")
}
