package transaction

type Propagation int8

type TransactionInfo struct {
	TimeOut           int32
	Name              string
	Propagation       Propagation
	LockRetryInternal int64
	LockRetryTimes    int64
}

const (

	/**
	 * The REQUIRED.
	 * The default propagation.
	 *
	 * <p>
	 * If transaction is existing, execute with current transaction,
	 * else execute with new transaction.
	 * </p>
	 *
	 * <p>
	 * The logic is similar to the following code:
	 * <code><pre>
	 *     if (tx == null) {
	 *         try {
	 *             tx = beginNewTransaction(); // begin new transaction, is not existing
	 *             Object rs = business.execute(); // execute with new transaction
	 *             commitTransaction(tx);
	 *             return rs;
	 *         } catch (Exception ex) {
	 *             rollbackTransaction(tx);
	 *             throw ex;
	 *         }
	 *     } else {
	 *         return business.execute(); // execute with current transaction
	 *     }
	 * </pre></code>
	 * </p>
	 */
	REQUIRED Propagation = iota

	/**
	 * The REQUIRES_NEW.
	 *
	 * <p>
	 * If transaction is existing, suspend it, and then execute business with new transaction.
	 * </p>
	 *
	 * <p>
	 * The logic is similar to the following code:
	 * <code><pre>
	 *     try {
	 *         if (tx != null) {
	 *             suspendedResource = suspendTransaction(tx); // suspend current transaction
	 *         }
	 *         try {
	 *             tx = beginNewTransaction(); // begin new transaction
	 *             Object rs = business.execute(); // execute with new transaction
	 *             commitTransaction(tx);
	 *             return rs;
	 *         } catch (Exception ex) {
	 *             rollbackTransaction(tx);
	 *             throw ex;
	 *         }
	 *     } finally {
	 *         if (suspendedResource != null) {
	 *             resumeTransaction(suspendedResource); // resume transaction
	 *         }
	 *     }
	 * </pre></code>
	 * </p>
	 */
	REQUIRES_NEW

	/**
	 * The NOT_SUPPORTED.
	 *
	 * <p>
	 * If transaction is existing, suspend it, and then execute business without transaction.
	 * </p>
	 *
	 * <p>
	 * The logic is similar to the following code:
	 * <code><pre>
	 *     try {
	 *         if (tx != null) {
	 *             suspendedResource = suspendTransaction(tx); // suspend current transaction
	 *         }
	 *         return business.execute(); // execute without transaction
	 *     } finally {
	 *         if (suspendedResource != null) {
	 *             resumeTransaction(suspendedResource); // resume transaction
	 *         }
	 *     }
	 * </pre></code>
	 * </p>
	 */
	NOT_SUPPORTED

	/**
	 * The SUPPORTS.
	 *
	 * <p>
	 * If transaction is not existing, execute without global transaction,
	 * else execute business with current transaction.
	 * </p>
	 *
	 * <p>
	 * The logic is similar to the following code:
	 * <code><pre>
	 *     if (tx != null) {
	 *         return business.execute(); // execute with current transaction
	 *     } else {
	 *         return business.execute(); // execute without transaction
	 *     }
	 * </pre></code>
	 * </p>
	 */
	SUPPORTS

	/**
	 * The NEVER.
	 *
	 * <p>
	 * If transaction is existing, throw exception,
	 * else execute business without transaction.
	 * </p>
	 *
	 * <p>
	 * The logic is similar to the following code:
	 * <code><pre>
	 *     if (tx != null) {
	 *         throw new TransactionException("existing transaction");
	 *     }
	 *     return business.execute(); // execute without transaction
	 * </pre></code>
	 * </p>
	 */
	NEVER

	/**
	 * The MANDATORY.
	 *
	 * <p>
	 * If transaction is not existing, throw exception,
	 * else execute business with current transaction.
	 * </p>
	 *
	 * <p>
	 * The logic is similar to the following code:
	 * <code><pre>
	 *     if (tx == null) {
	 *         throw new TransactionException("not existing transaction");
	 *     }
	 *     return business.execute(); // execute with current transaction
	 * </pre></code>
	 * </p>
	 */
	MANDATORY
)
