package transfer

type Canceler struct {
    Cancel func()
}