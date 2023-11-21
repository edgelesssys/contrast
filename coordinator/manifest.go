package main

type Manifest struct {
	Policies        map[string]string
	ReferenceValues ReferenceValues
}

type ReferenceValues struct {
	SNP SNPReferenceValues
}

type SNPReferenceValues struct{}
