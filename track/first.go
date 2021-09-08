package track

import (
	"errors"
	"fmt"
	"regexp"
	"regexp/syntax"
	"strings"
)

var _ Track = (*First)(nil)

type First struct {
	FixedLength bool
}

func NewTrackFirst(fixed bool) *First {
	return &First{FixedLength: fixed}
}

func (h *First) Write(card *GeneralCard) ([]byte, error) {
	generator, _ := NewGenerator(trackFirstPattern, &GeneratorArgs{
		Flags: syntax.Perl,
		CaptureGroupHandler: func(index int, name string, group *syntax.Regexp, generator Generator, args *GeneratorArgs) string {
			var raw string
			switch index {
			case 0:
				raw = card.FormatCode
			case 1:
				raw = card.PrimaryAccountNumber
			case 2:
				if len(card.Name) > 0 && h.FixedLength {
					raw = fmt.Sprintf("%-26.26s", card.Name)
				} else {
					raw = card.Name
				}
			case 3:
				if card.ExpirationDate != nil {
					raw = card.ExpirationDate.String()
				}
			case 4:
				raw = card.ServiceCode
			case 5:
				raw = card.DiscretionaryData
			}

			if len(raw) == 0 {
				return `^`
			}
			return raw
		},
	})

	rawTrack := generator.Generate()
	if matched, _ := regexp.MatchString(trackFirstPattern, rawTrack); !matched {
		return nil, errors.New("unable to create valid track data")
	}

	if len(rawTrack) > trackFirstMaxLength {
		return nil, errors.New("unable to create valid track data")
	}

	return []byte(rawTrack), nil
}

func (h *First) Read(raw []byte) (*GeneralCard, error) {
	if raw == nil || len(raw) > trackFirstMaxLength {
		return nil, errors.New("invalid track 1 format")
	}

	r, err := regexp.Compile(trackFirstPattern)
	if err != nil {
		return nil, err
	}

	if !r.MatchString(string(raw)) {
		return nil, errors.New("invalid track 1 format")
	}

	var card GeneralCard
	matches := r.FindStringSubmatch(string(raw))
	for index, val := range matches {
		value := strings.TrimSpace(val)
		if len(value) == 0 || value == "^" {
			continue
		}

		switch index {
		case 1: // Format Code
			card.FormatCode = value
		case 2: // Payment card number (PAN)
			card.PrimaryAccountNumber = value
			card.CardType = GetCardType(value)
		case 3: // Name (NM)
			card.Name = value
		case 4: // Expiration Date (ED)
			card.ExpirationDate, err = NewExpiryDate(value)
		case 5: // Service Code (SC)
			card.ServiceCode = value
		case 6: // Discretionary data (DD)
			card.DiscretionaryData = value
		}
	}

	return &card, err
}
