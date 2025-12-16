package cmd

import "github.com/huanfeng/apkhub-cli/internal/i18n"

// applyCommandLocalization updates command and flag descriptions after i18n is initialized.
func applyCommandLocalization() {
	// Root command metadata and flags.
	rootCmd.Short = i18n.T("cmd.root.short")
	rootCmd.Long = i18n.T("cmd.root.long")

	if flag := rootCmd.PersistentFlags().Lookup("config"); flag != nil {
		flag.Usage = i18n.T("flags.config")
	}
	if flag := rootCmd.PersistentFlags().Lookup("work-dir"); flag != nil {
		flag.Usage = i18n.T("flags.workDir")
	}
	if flag := rootCmd.PersistentFlags().Lookup("verbose"); flag != nil {
		flag.Usage = i18n.T("flags.verbose")
	}
	if flag := rootCmd.PersistentFlags().Lookup("debug"); flag != nil {
		flag.Usage = i18n.T("flags.debug")
	}
	if flag := rootCmd.PersistentFlags().Lookup("log-file"); flag != nil {
		flag.Usage = i18n.T("flags.logFile")
	}
	if flag := rootCmd.PersistentFlags().Lookup("no-color"); flag != nil {
		flag.Usage = i18n.T("flags.noColor")
	}
	if flag := rootCmd.PersistentFlags().Lookup("lang"); flag != nil {
		flag.Usage = i18n.T("flags.lang")
	}

	// Command descriptions.
	repoCmd.Short = i18n.T("cmd.repo.short")
	repoCmd.Long = i18n.T("cmd.repo.long")

	scanCmd.Short = i18n.T("cmd.scan.short")
	scanCmd.Long = i18n.T("cmd.scan.long")

	initCmd.Short = i18n.T("cmd.init.short")
	initCmd.Long = i18n.T("cmd.init.long")

	listCmd.Short = i18n.T("cmd.list.short")
	listCmd.Long = i18n.T("cmd.list.long")

	searchCmd.Short = i18n.T("cmd.search.short")
	searchCmd.Long = i18n.T("cmd.search.long")

	downloadCmd.Short = i18n.T("cmd.download.short")
	downloadCmd.Long = i18n.T("cmd.download.long")

	installCmd.Short = i18n.T("cmd.install.short")
	installCmd.Long = i18n.T("cmd.install.long")

	doctorCmd.Short = i18n.T("cmd.doctor.short")
	doctorCmd.Long = i18n.T("cmd.doctor.long")

	versionCmd.Short = i18n.T("cmd.version.short")
	versionCmd.Long = i18n.T("cmd.version.long")

	// Additional command metadata (description only).
	if addCmd != nil {
		addCmd.Short = i18n.T("cmd.repoAdd.short")
		addCmd.Long = i18n.T("cmd.repoAdd.long")
	}
	bucketCmd.Short = i18n.T("cmd.bucket.short")
	bucketCmd.Long = i18n.T("cmd.bucket.long")
	cacheCmd.Short = i18n.T("cmd.cache.short")
	cacheCmd.Long = i18n.T("cmd.cache.long")
	cleanCmd.Short = i18n.T("cmd.clean.short")
	cleanCmd.Long = i18n.T("cmd.clean.long")
	depsCmd.Short = i18n.T("cmd.deps.short")
	depsCmd.Long = i18n.T("cmd.deps.long")
	devicesCmd.Short = i18n.T("cmd.devices.short")
	devicesCmd.Long = i18n.T("cmd.devices.long")
	exportCmd.Short = i18n.T("cmd.export.short")
	exportCmd.Long = i18n.T("cmd.export.long")
	importCmd.Short = i18n.T("cmd.import.short")
	importCmd.Long = i18n.T("cmd.import.long")
	infoCmd.Short = i18n.T("cmd.info.short")
	infoCmd.Long = i18n.T("cmd.info.long")
	parseCmd.Short = i18n.T("cmd.parse.short")
	parseCmd.Long = i18n.T("cmd.parse.long")
	parserInfoCmd.Short = i18n.T("cmd.parserInfo.short")
	parserInfoCmd.Long = i18n.T("cmd.parserInfo.long")
	statsCmd.Short = i18n.T("cmd.stats.short")
	statsCmd.Long = i18n.T("cmd.stats.long")
	updateCmd.Short = i18n.T("cmd.update.short")
	updateCmd.Long = i18n.T("cmd.update.long")
	verifyCmd.Short = i18n.T("cmd.verify.short")
	verifyCmd.Long = i18n.T("cmd.verify.long")
}
