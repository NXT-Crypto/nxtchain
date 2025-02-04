package nxtblock

var TransactionPool = make(map[string]Transaction)

// * ADD TRANSACTION TO TRANSACTION POOL * //

func AddTransactionToPool(transaction Transaction) {
	TransactionPool[transaction.Hash] = transaction
}

// * REMOVE TRANSACTION FROM TRANSACTION POOL * //

func RemoveTransactionFromPool(transaction Transaction) {
	delete(TransactionPool, transaction.Hash)
}

// * GET TRANSACTION FROM TRANSACTION POOL * //

func GetTransactionFromPool(hash string) (Transaction, bool) {
	transaction, exists := TransactionPool[hash]
	return transaction, exists
}

// * GET ALL TRANSACTIONS FROM TRANSACTION POOL * //

func GetAllTransactionsFromPool() map[string]Transaction {
	return TransactionPool
}

// * GET TRANSACTION POOL SIZE * //

func GetTransactionPoolSize() int {
	return len(TransactionPool)
}
