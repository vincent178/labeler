package labeler

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"testing"
)

func loadPayload(name string) ([]byte, error) {
	file, err := os.Open("../test_data/" + name + "_payload")
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(file)
}

type TestCase struct {
	payloads       []string
	name           string
	config         LabelerConfigV1
	initialLabels  []string
	expectedLabels []string
}

func TestHandleEvent(t *testing.T) {

	// These all use the payload in payload files
	testCases := []TestCase{
		TestCase{
			payloads:       []string{"create_pr", "reopen_pr"},
			name:           "Empty config",
			config:         LabelerConfigV1{},
			initialLabels:  []string{"Fix"},
			expectedLabels: []string{"Fix"},
		},
		TestCase{
			payloads: []string{"create_pr", "reopen_pr"},
			name:     "Config with no rules",
			config: LabelerConfigV1{
				Version: 1,
				Labels: []LabelMatcher{
					LabelMatcher{
						Label: "WIP",
					},
				},
			},
			initialLabels:  []string{"Fix"},
			expectedLabels: []string{"Fix"},
		},
		TestCase{
			payloads: []string{"create_pr", "reopen_pr"},
			name:     "Add a label when not set and config matches",
			config: LabelerConfigV1{
				Version: 1,
				Labels: []LabelMatcher{
					LabelMatcher{
						Label: "WIP",
						Title: "^WIP:.*",
					},
				},
			},
			initialLabels:  []string{},
			expectedLabels: []string{"WIP"},
		},
		TestCase{
			payloads: []string{"create_pr", "reopen_pr"},
			name:     "Remove a label when set and config does not match",
			config: LabelerConfigV1{
				Version: 1,
				Labels: []LabelMatcher{
					LabelMatcher{
						Label: "Fix",
						Title: "Fix: .*",
					},
				},
			},
			initialLabels:  []string{"Fix"},
			expectedLabels: []string{},
		},
		TestCase{
			payloads: []string{"create_pr", "reopen_pr"},
			name:     "Respect a label when set, and not present in config",
			config: LabelerConfigV1{
				Version: 1,
				Labels: []LabelMatcher{
					LabelMatcher{
						Label: "Fix",
						Title: "^Fix.*",
					},
				},
			},
			initialLabels:  []string{"SomeLabel"},
			expectedLabels: []string{"SomeLabel"},
		},
		TestCase{
			payloads: []string{"create_pr", "reopen_pr"},
			name:     "A combination of all cases",
			config: LabelerConfigV1{
				Version: 1,
				Labels: []LabelMatcher{
					LabelMatcher{
						Label: "WIP",
						Title: "^WIP:.*",
					},
					LabelMatcher{
						Label: "ShouldRemove",
						Title: "^MEH.*",
					},
				},
			},
			initialLabels:  []string{"ShouldRemove", "ShouldRespect"},
			expectedLabels: []string{"WIP", "ShouldRespect"},
		},
		TestCase{
			payloads: []string{"create_pr", "reopen_pr"},
			name:     "Add a label with two conditions, both matching",
			config: LabelerConfigV1{
				Version: 1,
				Labels: []LabelMatcher{
					LabelMatcher{
						Label:     "WIP",
						Title:     "^WIP:.*",
						Mergeable: "False",
					},
				},
			},
			initialLabels:  []string{},
			expectedLabels: []string{"WIP"},
		},
		TestCase{
			payloads: []string{"create_pr", "reopen_pr"},
			name:     "Add a label with two conditions, one not matching (1)",
			config: LabelerConfigV1{
				Version: 1,
				Labels: []LabelMatcher{
					LabelMatcher{
						Label:     "WIP",
						Title:     "^WIP:.*",
						Mergeable: "True",
					},
				},
			},
			initialLabels:  []string{},
			expectedLabels: []string{},
		},
		TestCase{
			// covers evaluation order making a True in the last
			// condition, while previous ones are false
			payloads: []string{"create_pr", "reopen_pr"},
			name:     "Add a label with two conditions, one not matching (2)",
			config: LabelerConfigV1{
				Version: 1,
				Labels: []LabelMatcher{
					LabelMatcher{
						Label:     "WIP",
						Title:     "^DOES NOT MATCH:.*",
						Mergeable: "False",
					},
				},
			},
			initialLabels:  []string{},
			expectedLabels: []string{},
		},
		TestCase{
			payloads: []string{"small_pr"},
			name:     "Test the size_below rule",
			config: LabelerConfigV1{
				Version: 1,
				Labels: []LabelMatcher{
					LabelMatcher{
						Label:     "S",
						SizeBelow: "10",
					},
				},
			},
			initialLabels:  []string{},
			expectedLabels: []string{"S"},
		},
		TestCase{
			payloads: []string{"mid_pr"},
			name:     "Test the size_below and size_above rules",
			config: LabelerConfigV1{
				Version: 1,
				Labels: []LabelMatcher{
					LabelMatcher{
						Label:     "M",
						SizeAbove: "9",
						SizeBelow: "100",
					},
				},
			},
			initialLabels:  []string{},
			expectedLabels: []string{"M"},
		},
		TestCase{
			payloads: []string{"big_pr"},
			name:     "Test the size_above rule",
			config: LabelerConfigV1{
				Version: 1,
				Labels: []LabelMatcher{
					LabelMatcher{
						Label:     "L",
						SizeAbove: "100",
					},
				},
			},
			initialLabels:  []string{},
			expectedLabels: []string{"L"},
		},
		TestCase{
			payloads: []string{"small_pr"},
			name:     "Test the branch rule (matching)",
			config: LabelerConfigV1{
				Version: 1,
				Labels: []LabelMatcher{
					LabelMatcher{
						Label:  "Branch",
						Branch: "^srvaroa-patch.*",
					},
				},
			},
			initialLabels:  []string{},
			expectedLabels: []string{"Branch"},
		},
		TestCase{
			payloads: []string{"small_pr"},
			name:     "Test the branch rule (not matching)",
			config: LabelerConfigV1{
				Version: 1,
				Labels: []LabelMatcher{
					LabelMatcher{
						Label:  "Branch",
						Branch: "^does/not-match/*",
					},
				},
			},
			initialLabels:  []string{},
			expectedLabels: []string{},
		},
		TestCase{
			payloads: []string{"diff_pr"},
			name:     "Test the files rule",
			config: LabelerConfigV1{
				Version: 1,
				Labels: []LabelMatcher{
					LabelMatcher{
						Label: "Files",
						Files: []string{
							"^pkg/.*_test.go",
						},
					},
				},
			},
			initialLabels:  []string{},
			expectedLabels: []string{"Files"},
		},
		TestCase{
			payloads: []string{"small_pr"},
			name:     "Multiple conditions for the same tag function as OR",
			config: LabelerConfigV1{
				Version: 1,
				Labels: []LabelMatcher{
					LabelMatcher{
						Label:  "Branch",
						Branch: "^srvaroa-patch.*",
					},
					LabelMatcher{
						Label:  "Branch",
						Branch: "WONT MATCH",
					},
				},
			},
			initialLabels:  []string{},
			expectedLabels: []string{"Branch"},
		},
	}

	for _, tc := range testCases {
		for _, file := range tc.payloads {
			payload, err := loadPayload(file)
			if err != nil {
				t.Fatal(err)
			}

			fmt.Println(tc.name)
			l := NewTestLabeler(t, tc)
			err = l.HandleEvent("pull_request", &payload)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func NewTestLabeler(t *testing.T, tc TestCase) Labeler {
	return Labeler{
		FetchRepoConfig: func(owner, repoName string) (*LabelerConfigV1, error) {
			return &tc.config, nil
		},
		GetCurrentLabels: func(owner, repoName string, prNumber int) ([]string, error) {
			return tc.initialLabels, nil
		},
		ReplaceLabelsForPr: func(owner, repoName string, prNumber int, labels []string) error {
			sort.Strings(tc.expectedLabels)
			sort.Strings(labels)
			if reflect.DeepEqual(tc.expectedLabels, labels) {
				return nil
			}
			return fmt.Errorf("%s: Expecting %+v, got %+v",
				tc.name, tc.expectedLabels, labels)
		},
	}
}
