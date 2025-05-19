package types

type Signal interface {
    CalculateSignal() (float64, error)
}

type PublishOptions = func(Publisher)

func WithSignal(signal float64) PublishOptions {
    return func(p Publisher) {
        p.SetSignal(signal)
    }
}

func WithCapacity(capacity float64) PublishOptions {
    return func(p Publisher) {
        p.SetCapacity(capacity)
    }
}

func WithOverprovision(overProvision float64) PublishOptions {
    return func(p Publisher) {
        p.SetOverProvision(overProvision)
    }
}

type Publisher interface {
    SetSignal(signal float64)
    SetCapacity(capacity float64)
    SetOverProvision(op float64)
    Publish(opts ...PublishOptions)
}

type Capacity interface {
    Update(podCount int, signal float64)
    GetCapacityFromPodCount(podCount int) float64
    GetCapacityFromSignal(signal float64) float64
}
