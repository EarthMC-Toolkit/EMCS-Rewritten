package oapi

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type EventResponse struct {
	Event string
	Data  string
}

type Event string

const (
	EventNewDay Event = "NewDay"

	EventNationCreated     Event = "NationCreated"
	EventNationDeleted     Event = "NationDeleted"
	EventNationRenamed     Event = "NationRenamed"
	EventNationKingChanged Event = "NationKingChanged"
	EventNationMerged      Event = "NationMerged"

	EventTownCreated      Event = "TownCreated"
	EventTownDeleted      Event = "TownDeleted"
	EventTownRenamed      Event = "TownRenamed"
	EventTownMayorChanged Event = "TownMayorChanged"
	EventTownMerged       Event = "TownMerged"
	EventTownRuined       Event = "TownRuined"
	EventTownReclaimed    Event = "TownReclaimed"

	EventResidentJoinedTown Event = "ResidentJoinedTown"
	EventResidentLeftTown   Event = "ResidentLeftTown"
	EventTownJoinedNation   Event = "TownJoinedNation"
	EventTownLeftNation     Event = "TownLeftNation"

	EventShopSoldItem   Event = "ShopSoldItem"
	EventShopBoughtItem Event = "ShopBoughtItem"
	EventShopOutOfStock Event = "ShopOutOfStock"
	EventShopOutOfSpace Event = "ShopOutOfSpace"
	EventShopOutOfGold  Event = "ShopOutOfGold"
)

// The following events are able to be listened to by anyone with a valid bearer key.
var GLOBAL_EVENTS = []Event{
	EventNationCreated,
	EventNationDeleted,
	EventNationRenamed,
	EventNationKingChanged,
	EventNationMerged,

	EventTownCreated,
	EventTownDeleted,
	EventTownRenamed,
	EventTownMayorChanged,
	EventTownMerged,
	EventTownRuined,
	EventTownReclaimed,

	EventNewDay,
}

// The following events are 'priveleged' in that they are only sent to the relevent player.
var PRIVILEGED_EVENTS = []Event{
	EventTownJoinedNation,
	EventTownLeftNation,

	EventResidentJoinedTown,
	EventResidentLeftTown,

	EventShopSoldItem,
	EventShopBoughtItem,
	EventShopOutOfStock,
	EventShopOutOfSpace,
	EventShopOutOfGold,
}

func ListenToSSE(ctx context.Context, events []Event, authKey string) error {
	req, err := createReqWithAuth(ctx, authKey, events)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	return readSSE(resp.Body, func(r EventResponse) {
		fmt.Println("EVENT:", r.Event)
		fmt.Println("DATA:", r.Data)
	})
}

func createReqWithAuth(ctx context.Context, authKey string, events []Event) (*http.Request, error) {
	if authKey == "" {
		return nil, fmt.Errorf("empty OAPI auth key")
	}

	u, err := url.Parse(ENDPOINT_SSE)
	if err != nil {
		return nil, err
	}

	evNames := make([]string, len(events))
	for i, e := range events {
		evNames[i] = string(e)
	}

	q := u.Query()
	q.Set("listen", strings.Join(evNames, ","))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+authKey)
	req.Header.Set("Accept", "text/event-stream")

	return req, err
}

func readSSE(body io.ReadCloser, handler func(EventResponse)) error {
	defer body.Close()

	reader := bufio.NewReader(body)
	evt := EventResponse{}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "event:"):
			evt.Event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))

		case strings.HasPrefix(line, "data:"):
			evt.Data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))

			// full message ready
			handler(evt)
			evt = EventResponse{}
		}
	}
}
