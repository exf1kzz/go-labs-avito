package trip

type TripRepository struct {
	transactionManager transactionManager
}

func NewTripRepository(manager transactionManager) *TripRepository {
	return &TripRepository{transactionManager: manager}
}
