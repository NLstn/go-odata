package async

import "testing"

func TestJobRecordTableName(t *testing.T) {
	record := JobRecord{}
	if record.TableName() != "_odata_async_jobs" {
		t.Fatalf("expected table name _odata_async_jobs, got %q", record.TableName())
	}
}
