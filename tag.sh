#!/bin/bash
# 自动生成伪版本号的脚本示例
COMMIT=$(git rev-parse --short=12 HEAD)
TIMESTAMP=$(python3 - <<'PY'
import subprocess, datetime
iso = subprocess.check_output(["git","show","-s","--format=%cI","HEAD"], text=True).strip()
dt = datetime.datetime.fromisoformat(iso.replace("Z", "+00:00")).astimezone(datetime.timezone.utc)
print(dt.strftime("%Y%m%d%H%M%S"))
PY
)
VERSION="v0.0.0-${TIMESTAMP}-${COMMIT}"
echo $VERSION