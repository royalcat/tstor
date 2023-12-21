package torrent

import (
	"encoding/binary"
	"fmt"
	"log/slog"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
	"github.com/dgraph-io/badger/v4"
)

type PieceCompletionState byte

const (
	PieceNotComplete PieceCompletionState = 0
	PieceComplete    PieceCompletionState = 1<<8 - 1
)

func pieceCompletionState(i bool) PieceCompletionState {
	if i {
		return PieceComplete
	} else {
		return PieceNotComplete
	}
}

type badgerPieceCompletion struct {
	db *badger.DB
}

var _ storage.PieceCompletion = (*badgerPieceCompletion)(nil)

func NewBadgerPieceCompletion(dir string) (storage.PieceCompletion, error) {
	opts := badger.
		DefaultOptions(dir).
		WithLogger(badgerSlog{slog: slog.With("component", "piece-completion")})
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &badgerPieceCompletion{db}, nil
}

func pkToBytes(pk metainfo.PieceKey) []byte {
	key := make([]byte, len(pk.InfoHash.Bytes()))
	copy(key, pk.InfoHash.Bytes())
	binary.BigEndian.AppendUint32(key, uint32(pk.Index))
	return key
}

func (k *badgerPieceCompletion) Get(pk metainfo.PieceKey) (storage.Completion, error) {
	completion := storage.Completion{
		Ok: true,
	}
	err := k.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(pkToBytes(pk))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				completion.Ok = false
				return nil
			}

			return fmt.Errorf("getting value: %w", err)
		}

		valCopy, err := item.ValueCopy(nil)
		if err != nil {
			return fmt.Errorf("copying value: %w", err)
		}
		compl := PieceCompletionState(valCopy[0])

		completion.Ok = true
		switch compl {
		case PieceComplete:
			completion.Complete = true
		case PieceNotComplete:
			completion.Complete = false
		}

		return nil
	})
	return completion, err
}

func (me badgerPieceCompletion) Set(pk metainfo.PieceKey, b bool) error {
	if c, err := me.Get(pk); err == nil && c.Ok && c.Complete == b {
		return nil
	}

	return me.db.Update(func(txn *badger.Txn) error {
		return txn.Set(pkToBytes(pk), []byte{byte(pieceCompletionState(b))})
	})
}

func (k *badgerPieceCompletion) Delete(key string) error {
	return k.db.Update(
		func(txn *badger.Txn) error {
			return txn.Delete([]byte(key))
		})
}

func (me *badgerPieceCompletion) Close() error {
	return me.db.Close()
}

type badgerSlog struct {
	slog *slog.Logger
}

// Debugf implements badger.Logger.
func (log badgerSlog) Debugf(f string, a ...interface{}) {
	log.slog.Debug(f, a...)
}

// Errorf implements badger.Logger.
func (log badgerSlog) Errorf(f string, a ...interface{}) {
	log.slog.Error(f, a...)
}

// Infof implements badger.Logger.
func (log badgerSlog) Infof(f string, a ...interface{}) {
	log.slog.Info(f, a...)
}

// Warningf implements badger.Logger.
func (log badgerSlog) Warningf(f string, a ...interface{}) {
	log.slog.Warn(f, a...)
}

var _ badger.Logger = (*badgerSlog)(nil)
