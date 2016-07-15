# With tag: 1.0.0[-commits since tag][-dirty]
# Without existing tags: 4a4154a[-dirty]
version=$(git describe --tags --always --dirty)
# 2000-12-13 13:00:00
buildDate=$(date --utc "+%F %T")
go build -ldflags "-X 'main.Version=$version' -X 'main.BuildDate=$buildDate'"
