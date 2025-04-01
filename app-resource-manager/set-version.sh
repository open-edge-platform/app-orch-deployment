#!/usr/bin/env bash

# Copyright 2023 Intel Corp.
# SPDX-License-Identifier: Apache-2.0

# increment the supplied version string with an optional suffix
function get_next_version {
  local version=$1
  local new_suffix=${2:-}

  if [[ "$version" =~ ^v?([0-9]+)\.([0-9]+)\.([0-9]+)(\-?.*)$ ]]
  then
    major="${BASH_REMATCH[1]}"
    minor="${BASH_REMATCH[2]}"
    patch="${BASH_REMATCH[3]}"
    suffix="${BASH_REMATCH[4]}"

    # The "patch" value stays the same if the current version has a suffix and 
    # the requested version has no suffix or the requested suffix is alphabetiacally
    # greater than the existing suffix. This will result in:
    #   0.1.0-dev -> 0.1.0 -> 0.1.1-dev -> 0.1.1-rc1 -> 0.1.1 -> 0.1.2
    if [ -z "$new_suffix" ]
    then
      # no new_suffix, if there is no suffix, increment patch otherwise leave it as is
      if [ -z "$suffix" ]
      then
        patch=$((patch+1))
      fi
    else
      # new_suffix exists, if not greater than suffix, increment patch otherwise leave it as is
      if [ -z "$suffix" ] || [[ ! "$new_suffix" > "$suffix" ]]
      then
        patch=$((patch+1))
      fi
    fi

    echo "$major.$minor.$patch$new_suffix"
    return
  fi

  # default to 0.1.0 with the supplied new_suffix if no release tag can be found
  echo "0.1.0$new_suffix"
}

if [ -n "$1" ]
then
  new_version="$1"
else
  echo "ERROR: missing required version parameter"
  exit 1
fi

if [ -n "$2" ]
then
  new_version="$new_version$2"
  new_suffix=$2
else
  new_suffix=
fi

# Update VERSION file if present
if [ -f "VERSION" ]
then
  echo "  - update VERSION to ${new_version}"
  echo "${new_version}" > "VERSION"
fi

# Update Umbrella Chart versioned repos
if [ -f ".chartver.yaml" ] && [ -n "$(yq '.release // "" ' .chartver.yaml)" ]
then
  chart="$(yq .release .chartver.yaml)"
  echo "  - update $chart/Chart.yaml .version to ${new_dev_version}"
  yq -i ".version = \"${new_version}\"" "$chart/Chart.yaml"
  yq -i ".appVersion = \"${new_version}\"" "$chart/Chart.yaml"
  echo "  - update $chart/Chart.yaml remove any build annotations"
  yq eval -i 'del(.annotations.revision)' "$chart/Chart.yaml"
  yq eval -i 'del(.annotations.created)' "$chart/Chart.yaml"
fi

# Update Charts specified in the devCharts list in .chartver.yaml
if [ -f ".chartver.yaml" ] && [ -n "$(yq '.devCharts[] // "" ' .chartver.yaml)" ]
then
  for chart in $(yq .devCharts[] .chartver.yaml)
  do
    chart_version=$(yq .version "$chart/Chart.yaml")
    chart_new_version=$(get_next_version "$chart_version" "$new_suffix")
    echo "  - update $chart/Chart.yaml .version to ${chart_new_version}"
    yq -i ".version = \"${chart_new_version}\"" "$chart/Chart.yaml"
    echo "  - update $chart/Chart.yaml .appVersion to ${new_version}"
    yq -i ".appVersion = \"${new_version}\"" "$chart/Chart.yaml"
    echo "  - update $chart/Chart.yaml remove any build annotations"
    yq eval -i 'del(.annotations.revision)' "$chart/Chart.yaml"
    yq eval -i 'del(.annotations.created)' "$chart/Chart.yaml"
  done
fi
