package importing

func canReuseCommitResult(status string) bool {
	return status == BatchStatusCommitting || status == BatchStatusCommitted
}

func validateCommitStatus(status string) error {
	if status == BatchStatusValidated || canReuseCommitResult(status) {
		return nil
	}
	return newBatchStatusActionError("commit", status, BatchStatusValidated, BatchStatusCommitting, BatchStatusCommitted)
}

func canWorkerProcessCommit(status string) bool {
	return status == BatchStatusValidated || status == BatchStatusCommitting
}
