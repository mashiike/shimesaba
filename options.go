package shimesaba

type Options struct {
	dryRun   bool
	backfill int
}

//DryRunOption is an option to output the calculated error budget as standard without posting it to Mackerel.
func DryRunOption(dryRun bool) func(*Options) {
	return func(opt *Options) {
		opt.dryRun = dryRun
	}
}

//BackfillOption specifies how many points of data to calculate retroactively from the current time.
func BackfillOption(count int) func(*Options) {
	return func(opt *Options) {
		opt.backfill = count
	}
}
