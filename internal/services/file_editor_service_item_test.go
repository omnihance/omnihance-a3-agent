package services

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/project-agonyl/agonyl-utils-go/dropfile"
	"github.com/project-agonyl/agonyl-utils-go/itemfile"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestReadClientItemFileBytes(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)

	tests := []struct {
		name     string
		write    func(*bytes.Buffer) error
		read     func([]byte) ([]itemfile.Item, error)
		expected itemfile.Item
	}{
		{
			name: "it0",
			write: func(buf *bytes.Buffer) error {
				data := itemfile.IT0File{{ItemCodeBase: 2, Row: 0, Slot: 3, Type: 1, NPCPrice: 100}}
				copy(data[0].Name[:], "Sword")
				return itemfile.WriteIT0(buf, data)
			},
			read: service.ReadClientIT0FileBytes,
			expected: itemfile.Item{
				ItemCode:  2048,
				SlotIndex: 3,
				ItemName:  "Sword",
				Itemtype:  1,
				NPCPrice:  100,
			},
		},
		{
			name: "it1",
			write: func(buf *bytes.Buffer) error {
				data := itemfile.IT1File{{Type: 4, Row: 0, NPCPrice: 200}}
				copy(data[0].Name[:], "Potion")
				return itemfile.WriteIT1(buf, data)
			},
			read: service.ReadClientIT1FileBytes,
			expected: itemfile.Item{
				ItemCode:  4096,
				SlotIndex: 8,
				ItemName:  "Potion",
				Itemtype:  4,
				NPCPrice:  200,
			},
		},
		{
			name: "it2",
			write: func(buf *bytes.Buffer) error {
				data := itemfile.IT2File{{Type: 5, Row: 0, NPCPrice: 300, Class: 1}}
				copy(data[0].Name[:], "Skill")
				return itemfile.WriteIT2(buf, data)
			},
			read: service.ReadClientIT2FileBytes,
			expected: itemfile.Item{
				ItemCode: 5120,
				ItemName: "Skill",
				Itemtype: 5,
				NPCPrice: 300,
			},
		},
		{
			name: "it3",
			write: func(buf *bytes.Buffer) error {
				data := itemfile.IT3File{{Type: 6, Row: 0, NPCPrice: 400}}
				copy(data[0].Name[:], "Quest Item")
				return itemfile.WriteIT3(buf, data)
			},
			read: service.ReadClientIT3FileBytes,
			expected: itemfile.Item{
				ItemCode: 6144,
				ItemName: "Quest Item",
				Itemtype: 6,
				NPCPrice: 400,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf bytes.Buffer
			require.NoError(t, test.write(&buf))

			items, err := test.read(buf.Bytes())
			require.NoError(t, err)
			require.Len(t, items, 1)
			require.Equal(t, test.expected.ItemCode, items[0].ItemCode)
			require.Equal(t, test.expected.SlotIndex, items[0].SlotIndex)
			require.Equal(t, test.expected.ItemName, items[0].ItemName)
			require.Equal(t, test.expected.Itemtype, items[0].Itemtype)
			require.Equal(t, test.expected.NPCPrice, items[0].NPCPrice)
		})
	}
}

func TestReadClientItemFileBytesTruncated(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)

	_, err := service.ReadClientIT1FileBytes([]byte{1, 2, 3})
	require.Error(t, err)
}

func TestDropFileDataRoundTrip(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)
	path := filepath.Join(t.TempDir(), "100.itm")

	expected := dropfile.DropFile{
		{ItemID: 100, DropRate: 75, DropGroup: 1},
		{ItemID: 0x4001, DropRate: 200, DropGroup: 2},
		{ItemID: dropfile.EmptyItemID},
	}

	require.NoError(t, service.WriteDropFileData(path, expected))
	actual, err := service.ReadDropFileData(path)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestReadDropFileDataEmpty(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)
	path := filepath.Join(t.TempDir(), "empty.itm")
	require.NoError(t, os.WriteFile(path, nil, 0644))

	actual, err := service.ReadDropFileData(path)
	require.NoError(t, err)
	require.NotNil(t, actual)
	require.Empty(t, actual)
}

func TestReadDropFileDataTruncated(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)
	path := filepath.Join(t.TempDir(), "bad.itm")
	require.NoError(t, os.WriteFile(path, []byte{1, 2, 3}, 0644))

	_, err := service.ReadDropFileData(path)
	require.Error(t, err)
}
