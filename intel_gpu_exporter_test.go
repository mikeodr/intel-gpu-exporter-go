package main

import (
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestParseMetric(t *testing.T) {
	tests := []struct {
		name      string
		record    []string
		expected  IntelTopStats
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid input",
			// RCS %,RCS se,RCS wa,BCS %,BCS se,BCS wa,VCS %,VCS se,VCS wa,VECS %,VECS se,VECS wa
			record: []string{"1000.123", "95.1230", "500.23", "80.5", "3.2", "12.3", "13.2", "23.5", "23.3", "12.2", "10.3", "6.3", "5.5", "90.1", "12.3", "10.2"},
			expected: IntelTopStats{
				FreqMhzRequested: 1000.123,
				FreqMhzActual:    95.1230,
				IRQPerSec:        500.23,
				Rc6Percent:       80.5,
				Engine: map[string]IntelEngine{
					"RCS":  {BusyPercent: 3.2, SemaPercent: 12.3, WaitPercent: 13.2},
					"BCS":  {BusyPercent: 23.5, SemaPercent: 23.3, WaitPercent: 12.2},
					"VCS":  {BusyPercent: 10.3, SemaPercent: 6.3, WaitPercent: 5.5},
					"VECS": {BusyPercent: 90.1, SemaPercent: 12.3, WaitPercent: 10.2},
				},
			},
			expectErr: false,
		},
		{
			name:      "InvalidNumberOfFields",
			record:    []string{"1000", "950"}, // too few fields
			expectErr: true,
			errMsg:    "unexpected EOF",
		},
		{
			name:      "NonNumericField",
			record:    []string{"1000", "abc", "500", "80.5", "3.2", "0.0", "0.0", "0.0", "0.0", "0.0", "0.0", "0.0", "0.0", "0.0", "0.0", "0.0"},
			expectErr: true,
			errMsg:    `error parsing field 1 \(abc\): .*`,
		},
		{
			name:      "EmptyInput",
			record:    []string{},
			expectErr: true,
			errMsg:    "unexpected EOF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			s, err := parseMetric(tt.record)
			if tt.expectErr {
				c.Assert(err, qt.ErrorMatches, tt.errMsg)
			} else {
				c.Assert(err, qt.IsNil)
				c.Assert(s, qt.DeepEquals, tt.expected)
			}
		})
	}
}

func TestReadMetrics(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		name        string
		input       string
		expected    []IntelTopStats
		description string
	}{
		// Valid data tests
		{
			name: "ValidSingleRecord",
			input: `Freq MHz req,Freq MHz act,IRQ /s,RC6 %,RCS %,RCS se,RCS wa,BCS %,BCS se,BCS wa,VCS %,VCS se,VCS wa,VECS %,VECS se,VECS wa
1200.0,1150.0,500.0,85.5,10.2,5.1,2.3,15.4,7.8,3.2,8.9,4.5,1.8,12.7,6.3,2.9`,
			expected: []IntelTopStats{
				{
					FreqMhzRequested: 1200.0,
					FreqMhzActual:    1150.0,
					IRQPerSec:        500.0,
					Rc6Percent:       85.5,
					Engine: map[string]IntelEngine{
						"RCS":  {BusyPercent: 10.2, SemaPercent: 5.1, WaitPercent: 2.3},
						"BCS":  {BusyPercent: 15.4, SemaPercent: 7.8, WaitPercent: 3.2},
						"VCS":  {BusyPercent: 8.9, SemaPercent: 4.5, WaitPercent: 1.8},
						"VECS": {BusyPercent: 12.7, SemaPercent: 6.3, WaitPercent: 2.9},
					},
				},
			},
		},
		{
			name: "MultipleRecords",
			input: `Freq MHz req,Freq MHz act,IRQ /s,RC6 %,RCS %,RCS se,RCS wa,BCS %,BCS se,BCS wa,VCS %,VCS se,VCS wa,VECS %,VECS se,VECS wa
1200.0,1150.0,500.0,85.5,10.2,5.1,2.3,15.4,7.8,3.2,8.9,4.5,1.8,12.7,6.3,2.9
1300.0,1250.0,600.0,90.0,20.5,10.2,4.6,25.8,15.6,6.4,18.8,9.0,3.6,25.4,12.6,5.8`,
			expected: []IntelTopStats{
				{
					FreqMhzRequested: 1200.0,
					FreqMhzActual:    1150.0,
					IRQPerSec:        500.0,
					Rc6Percent:       85.5,
					Engine: map[string]IntelEngine{
						"RCS":  {BusyPercent: 10.2, SemaPercent: 5.1, WaitPercent: 2.3},
						"BCS":  {BusyPercent: 15.4, SemaPercent: 7.8, WaitPercent: 3.2},
						"VCS":  {BusyPercent: 8.9, SemaPercent: 4.5, WaitPercent: 1.8},
						"VECS": {BusyPercent: 12.7, SemaPercent: 6.3, WaitPercent: 2.9},
					},
				},
				{
					FreqMhzRequested: 1300.0,
					FreqMhzActual:    1250.0,
					IRQPerSec:        600.0,
					Rc6Percent:       90.0,
					Engine: map[string]IntelEngine{
						"RCS":  {BusyPercent: 20.5, SemaPercent: 10.2, WaitPercent: 4.6},
						"BCS":  {BusyPercent: 25.8, SemaPercent: 15.6, WaitPercent: 6.4},
						"VCS":  {BusyPercent: 18.8, SemaPercent: 9.0, WaitPercent: 3.6},
						"VECS": {BusyPercent: 25.4, SemaPercent: 12.6, WaitPercent: 5.8},
					},
				},
			},
		},
		// Edge case tests
		{
			name:        "HeaderOnly",
			input:       `Freq MHz req,Freq MHz act,IRQ /s,RC6 %,RCS %,RCS se,RCS wa,BCS %,BCS se,BCS wa,VCS %,VCS se,VCS wa,VECS %,VECS se,VECS wa`,
			expected:    []IntelTopStats{},
			description: "Should handle header-only input",
		},
		{
			name:        "EmptyInput",
			input:       ``,
			expected:    []IntelTopStats{},
			description: "Should handle empty input",
		},
		// Error handling tests
		{
			name: "IncompleteRecord",
			input: `Freq MHz req,Freq MHz act,IRQ /s,RC6 %,RCS %,RCS se,RCS wa,BCS %,BCS se,BCS wa,VCS %,VCS se,VCS wa,VECS %,VECS se,VECS wa
1200.0,1150.0,500.0,85.5,10.2`,
			expected:    []IntelTopStats{},
			description: "Should skip incomplete records",
		},
		{
			name: "InvalidNumberFormat",
			input: `Freq MHz req,Freq MHz act,IRQ /s,RC6 %,RCS %,RCS se,RCS wa,BCS %,BCS se,BCS wa,VCS %,VCS se,VCS wa,VECS %,VECS se,VECS wa
abc,1150.0,500.0,85.5,10.2,5.1,2.3,15.4,7.8,3.2,8.9,4.5,1.8,12.7,6.3,2.9`,
			expected:    []IntelTopStats{},
			description: "Should skip records with invalid number format",
		},
		{
			name: "MixedValidAndInvalidRecords",
			input: `Freq MHz req,Freq MHz act,IRQ /s,RC6 %,RCS %,RCS se,RCS wa,BCS %,BCS se,BCS wa,VCS %,VCS se,VCS wa,VECS %,VECS se,VECS wa
1200.0,1150.0,500.0,85.5,10.2,5.1,2.3,15.4,7.8,3.2,8.9,4.5,1.8,12.7,6.3,2.9
abc,1150.0,500.0,85.5,10.2,5.1,2.3,15.4,7.8,3.2,8.9,4.5,1.8,12.7,6.3,2.9
1300.0,1250.0,600.0,90.0,20.5,10.2,4.6,25.8,15.6,6.4,18.8,9.0,3.6,25.4,12.6,5.8`,
			expected: []IntelTopStats{
				{
					FreqMhzRequested: 1200.0,
					FreqMhzActual:    1150.0,
					IRQPerSec:        500.0,
					Rc6Percent:       85.5,
					Engine: map[string]IntelEngine{
						"RCS":  {BusyPercent: 10.2, SemaPercent: 5.1, WaitPercent: 2.3},
						"BCS":  {BusyPercent: 15.4, SemaPercent: 7.8, WaitPercent: 3.2},
						"VCS":  {BusyPercent: 8.9, SemaPercent: 4.5, WaitPercent: 1.8},
						"VECS": {BusyPercent: 12.7, SemaPercent: 6.3, WaitPercent: 2.9},
					},
				},
			},
			description: "Should process valid records and skip invalid ones",
		},
		{
			name: "IncompleteRecords",
			input: `
			1200.0,1150.0,500.0,85.5,10.2`,
			expected:    []IntelTopStats{},
			description: "Should skip incomplete records and records with leading newline",
		},
	}

	for _, tt := range tests {
		c.Run(tt.name, func(c *qt.C) {
			reader := strings.NewReader(tt.input)
			results := make([]IntelTopStats, 0)

			for stats := range readMetrics(reader) {
				results = append(results, stats)
			}

			// Determine expected count
			expectedCount := len(tt.expected)

			c.Assert(len(results), qt.Equals, expectedCount, qt.Commentf(tt.description))

			// Always check deep equality since all tests now have expected values
			c.Assert(results, qt.DeepEquals, tt.expected)
		})
	}
}

func TestReadMetricsEarlyBreak(t *testing.T) {
	c := qt.New(t)

	input := `Freq MHz req,Freq MHz act,IRQ /s,RC6 %,RCS %,RCS se,RCS wa,BCS %,BCS se,BCS wa,VCS %,VCS se,VCS wa,VECS %,VECS se,VECS wa
1200.0,1150.0,500.0,85.5,10.2,5.1,2.3,15.4,7.8,3.2,8.9,4.5,1.8,12.7,6.3,2.9
1300.0,1250.0,600.0,90.0,20.5,10.2,4.6,25.8,15.6,6.4,18.8,9.0,3.6,25.4,12.6,5.8
1400.0,1350.0,700.0,95.0,30.8,15.3,6.9,35.2,23.4,9.6,28.7,13.5,5.4,35.1,18.9,8.7`

	reader := strings.NewReader(input)
	results := make([]IntelTopStats, 0)
	count := 0

	for stats := range readMetrics(reader) {
		results = append(results, stats)
		count++
		if count >= 2 {
			break // Early break to test iterator stopping
		}
	}

	c.Assert(len(results), qt.Equals, 2)
	c.Assert(results[0].FreqMhzRequested, qt.Equals, 1200.0)
	c.Assert(results[1].FreqMhzRequested, qt.Equals, 1300.0)
}

func BenchmarkReadMetrics(b *testing.B) {
	input := `Freq MHz req,Freq MHz act,IRQ /s,RC6 %,RCS %,RCS se,RCS wa,BCS %,BCS se,BCS wa,VCS %,VCS se,VCS wa,VECS %,VECS se,VECS wa
1200.0,1150.0,500.0,85.5,10.2,5.1,2.3,15.4,7.8,3.2,8.9,4.5,1.8,12.7,6.3,2.9
1300.0,1250.0,600.0,90.0,20.5,10.2,4.6,25.8,15.6,6.4,18.8,9.0,3.6,25.4,12.6,5.8`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(input)
		for range readMetrics(reader) {
			// Consume all records
		}
	}
}
