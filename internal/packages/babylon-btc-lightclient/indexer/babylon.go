package indexer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cosmostation/cvms/internal/common/types"
	"github.com/pkg/errors"
)

// FilterEvents filters events of type "babylon.btclightclient.v1.EventBTCHeaderInserted"
// and events of type "message" where an attribute with key "action" has the value "/babylon.btclightclient.v1.MsgInsertHeaders"
// case1 ) https://www.mintscan.io/babylon-testnet/tx/821DC4CD513A988BBFFE9DF7717593684798F8DF12D8E9A2373C98217A8FA806?sector=logs
// case2 ) https://www.mintscan.io/babylon-testnet/tx/A2F002F01AEFC23B01171D438950AD93B5418230DFAD56A60A2F54983ADB2F83?sector=logs
const (
	InsertHeadersTypeURL = "/babylon.btclightclient.v1.MsgInsertHeaders"
	BTCRollForwardEvent  = "babylon.btclightclient.v1.EventBTCRollForward"
	BTCRollbackEvent     = "babylon.btclightclient.v1.EventBTCRollBack"
)

// TODO: add return err
func filterBTCLightClientEvents(events []types.BlockEvent) ([]BTCInsertEvent, error) {
	bieList := make([]BTCInsertEvent, 0)
	btcHeaderList := make([]BTCHeader, 0)
	reporter := ""
	found := false

	for idx, e := range events {
		if e.TypeName == "message" {
			for _, attr := range e.Attributes {
				if attr.Key == "action" && attr.Value == InsertHeadersTypeURL {
					// if already found, we need reinit for new reporter
					if found {
						btcHeaderList = make([]BTCHeader, 0)
						reporter = ""
					}

					found = true
				}

				if attr.Key == "sender" && found {
					reporter = attr.Value
				}
			}
		}

		if found {
			if (e.TypeName == BTCRollForwardEvent) || (e.TypeName == BTCRollbackEvent) {
				for _, attr := range e.Attributes {
					if attr.Key == "header" {
						btcHeader, err := ParseBTCHeader(attr.Value)
						if err != nil {
							return nil, errors.Wrap(err, "failed to parse btc header event")
						}
						btcHeader.EventType = ExtractEventType(e.TypeName)
						btcHeaderList = append(btcHeaderList, btcHeader)
					}
				}
			}
		}

		if idx == (len(events) - 1) {
			// finally add
			bieList = append(bieList, BTCInsertEvent{
				ReporterAddress: reporter,
				BTCHeaders:      btcHeaderList,
			})
		}
	}

	// not found anyone return nil
	if !found {
		return nil, nil
	}

	return bieList, nil
}

type BTCInsertEventSummary struct {
	skip            bool
	BlockHeight     int64
	BTCInsertEvents []BTCInsertEvent
}

type BTCInsertEvent struct {
	ReporterAddress string
	BTCHeaders      []BTCHeader
}

// ToHeadersStringSlice converts BTCHeaders to a slice of strings
func (e BTCInsertEvent) ToHeadersStringSlice() string {
	var headers []string
	for _, header := range e.BTCHeaders {
		headers = append(headers, header.String())
	}

	return strings.Join(headers, "\n")
}

type BTCHeader struct {
	EventType string `json:"event_type"`
	Header    string `json:"header"`
	Hash      string `json:"hash"`
	Height    int64  `json:"height"`
	Work      string `json:"work"`
}

// String method for BTCInsertEvent
func (h BTCHeader) String() string {
	return fmt.Sprintf("EventType: %s, Header: %s, Hash: %s, Height: %d, Work: %s",
		h.EventType, h.Header, h.Hash, h.Height, h.Work)
}

// Convert Go struct to JSON
func (h BTCHeader) MustMarshalJSON() []byte {
	jsonData, err := json.Marshal(h)
	if err != nil {
		return []byte("unexpected")
	}

	return jsonData
}

// ParseBTCHeader parses the "value" field into BTCHeader
func ParseBTCHeader(value string) (BTCHeader, error) {
	var btcHeader BTCHeader
	err := json.Unmarshal([]byte(value), &btcHeader)
	if err != nil {
		return BTCHeader{}, err
	}
	return btcHeader, nil
}

// ExtractEventType extracts the event type name from a fully qualified event type string
func ExtractEventType(eventType string) string {
	parts := strings.Split(eventType, ".")
	if len(parts) == 0 {
		return eventType // Return original string if split fails
	}
	return parts[len(parts)-1]
}
