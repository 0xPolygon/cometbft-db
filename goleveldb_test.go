package db

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

func TestGoLevelDBNewGoLevelDB(t *testing.T) {
	name := fmt.Sprintf("test_%x", randStr(12))
	defer cleanupDBDir("", name)

	// Test we can't open the db twice for writing
	wr1, err := NewGoLevelDB(name, "")
	require.Nil(t, err)
	_, err = NewGoLevelDB(name, "")
	require.NotNil(t, err)
	err = wr1.Close() // Close the db to release the lock
	require.Nil(t, err)

	// Test we can open the db twice for reading only
	ro1, err := NewGoLevelDBWithOpts(name, "", &opt.Options{ReadOnly: true})
	require.Nil(t, err)
	defer ro1.Close()
	ro2, err := NewGoLevelDBWithOpts(name, "", &opt.Options{ReadOnly: true})
	require.Nil(t, err)
	defer ro2.Close()
}

func TestGoLevelDBIteratorWithOpts(t *testing.T) {
	name := fmt.Sprintf("test_%x", randStr(12))
	defer cleanupDBDir("", name)

	db, err := NewGoLevelDB(name, "")
	require.NoError(t, err)
	defer db.Close()

	for i := byte(0); i < 10; i++ {
		require.NoError(t, db.Set([]byte{i}, []byte{i}))
	}

	check := func(t *testing.T, itr Iterator, want []byte) {
		t.Helper()
		defer itr.Close()
		var got []byte
		for ; itr.Valid(); itr.Next() {
			got = append(got, itr.Key()[0])
		}
		require.NoError(t, itr.Error())
		require.Equal(t, want, got)
	}

	t.Run("forward DontFillCache=true", func(t *testing.T) {
		itr, err := db.IteratorWithOpts(nil, nil, &ReadOptions{DontFillCache: true})
		require.NoError(t, err)
		check(t, itr, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	})

	t.Run("reverse DontFillCache=true", func(t *testing.T) {
		itr, err := db.ReverseIteratorWithOpts(nil, nil, &ReadOptions{DontFillCache: true})
		require.NoError(t, err)
		check(t, itr, []byte{9, 8, 7, 6, 5, 4, 3, 2, 1, 0})
	})

	t.Run("nil opts equivalent to plain Iterator", func(t *testing.T) {
		itr, err := db.IteratorWithOpts(nil, nil, nil)
		require.NoError(t, err)
		check(t, itr, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	})

	t.Run("helper falls back when interface is asserted", func(t *testing.T) {
		// Sanity: GoLevelDB satisfies DBIteratorWithOpts so the helper should call into it.
		_, ok := DB(db).(DBIteratorWithOpts)
		require.True(t, ok)

		itr, err := IteratorWithOpts(db, nil, nil, &ReadOptions{DontFillCache: true})
		require.NoError(t, err)
		check(t, itr, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	})
}

func BenchmarkGoLevelDBRandomReadsWrites(b *testing.B) {
	name := fmt.Sprintf("test_%x", randStr(12))
	db, err := NewGoLevelDB(name, "")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		err = db.Close()
		require.NoError(b, err)
		cleanupDBDir("", name)
	}()

	benchmarkRandomReadsWrites(b, db)
}
