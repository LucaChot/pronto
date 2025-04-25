package central

import (
	"errors"
	"github.com/LucaChot/pronto/src/alias"
)

type CtlAliasTable struct {
    at          *alias.AliasTable
    n           int
    idxs        []int
    samples     int
    maxSet      bool
}

func NewCtlAliasTable(weights []float64) (*CtlAliasTable, error) {
    nw := len(weights)
    if nw == 0 {
        return nil, errors.New("weights slice cannot be empty")
    }

    at, err := alias.New(weights)
    if err != nil {
        return nil, err
    }

	return &CtlAliasTable{
        at: at,
        n: nw,
    }, nil
}

func (cat *CtlAliasTable) WithIndxs(idxs []int) (error) {
    if idxs != nil {
        errors.New("idxs slice already set")
    }
    ni := len(idxs)
    if ni == 0 {
        errors.New("idxs slice cannot be empty")
    }
    if ni != cat.n {
        errors.New("weights slice and idxs slice must be the same length")
    }
    cat.idxs = idxs
    return nil
}

func (cat *CtlAliasTable) WithMaxSamples(samples int) (error) {
    if cat.maxSet {
        errors.New("max samples can only be set once")
    }
    if samples <= 0 {
        errors.New("max samples must be greater than 0")
    }
    cat.samples = samples
    cat.maxSet = true
    return nil
}

func (cat *CtlAliasTable) Sample() (int, error) {
    if cat.maxSet {
        if cat.samples <= 0 {
            return -1, errors.New("no more samples remaining")
        }
    }
    idx, err := cat.at.Sample()
    if err != nil {
        return idx, err
    }
    cat.samples -= 1
    if cat.idxs != nil {
        return cat.idxs[idx], nil
    }
    return idx, nil
}

func (cat *CtlAliasTable) SamplesLeft() (bool, error) {
    if cat.maxSet {
        if cat.samples <= 0 {
            return false, nil
        }
        return true, nil
    }
    return true, errors.New("max samples not set")
}
