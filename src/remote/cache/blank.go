package cache

import "log"

type StaticInformer struct {}

func NewStaticInformer() Informer {
    log.Print("created static informer")
    return &StaticInformer{}
}

func (ci *StaticInformer) SetOnChange(onChange func(count int)) {}

func (ci *StaticInformer) Start() {}
