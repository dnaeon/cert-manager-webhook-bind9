#!/usr/bin/env bash

# set -e

TMPDIR=${TMPDIR:-"/tmp"}

# Removes an ACME Challenge TXT record
#
# $1: DNS Server
# $2: Zone name
# $3: Path to TSIG key
# $4: TTL
# $5: Token
function _remove_acme_challenge_record() {
    local _dns_server="${1}"
    local _zone_name="${2}"
    local _tsig_key="${3}"
    local _ttl="${4}"
    local _token="${5}"
    local _script=$( mktemp add-ns-record.XXXXXX )

    cat > "${_script}" <<__EOF__
debug yes
server ${_dns_server}
zone ${_zone_name}
update delete _acme-challenge.${_zone_name} TXT ${_token}
send
__EOF__

    nsupdate -k "${_tsig_key}" -v "${_script}"
}

_remove_acme_challenge_record $*
