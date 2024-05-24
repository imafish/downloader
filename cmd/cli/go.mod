module main

go 1.22.3

require downloader v1.0.0

replace downloader => ../..

require agent v1.0.0

require internal/utils v1.0.0 // indirect

replace agent => ../../agent

replace internal/utils => ../../internal/utils
