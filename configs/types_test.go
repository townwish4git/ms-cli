package configs

import "testing"

func TestDefaultConfigSetsTrainSyncTuningDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Training.RsyncCompress {
		t.Fatalf("expected rsync compression to be disabled by default")
	}
	if !cfg.Training.RsyncRespectGitIgnore {
		t.Fatalf("expected /train sync to respect .gitignore by default")
	}
	if cfg.Training.SyncParallelism != 0 {
		t.Fatalf("expected sync_parallelism default to use auto mode, got %d", cfg.Training.SyncParallelism)
	}
	if len(cfg.Training.Exclude) == 0 {
		t.Fatalf("expected built-in train excludes")
	}
}
