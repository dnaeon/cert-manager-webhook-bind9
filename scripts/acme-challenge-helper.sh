#!/usr/bin/env bash
#
# A helper script which uses nsupdate(1) to create and delete TXT
# records as part of ACME DNS-01 challenge

set -e

# Just in case it's not setup on the running environment
TMPDIR=${TMPDIR:-"/tmp"}

_SCRIPT_NAME="${0##*/}"

# Prints the usage of the sript
function _usage() {
    echo "${_SCRIPT_NAME} [create|delete] <zone> <tsig-key> <ttl> <token>"
    exit 64  # EX_USAGE
}

# Adds a new ACME Challenge TXT record
#
# $1: Zone name
# $2: Path to TSIG key
# $3: TTL
# $4: Token / Key
function _add_acme_challenge_record() {
    local _zone_name="${1}"
    local _tsig_key="${2}"
    local _ttl="${3}"
    local _token="${4}"
    local _script=$( mktemp delete-acme-challenge.XXXXXX )

    cat > "${_script}" <<__EOF__
debug yes
server ${_dns_server}
zone ${_zone_name}
update add _acme-challenge.${_zone_name} ${_ttl} TXT ${_token}
send
__EOF__

    nsupdate -k "${_tsig_key}" -v "${_script}"
    rm -f "${_script}"
}

# Removes an ACME Challenge TXT record
#
# $1: Zone name
# $2: Path to TSIG key
# $3: TTL
# $4: Token / Key
function _delete_acme_challenge_record() {
    local _zone_name="${1}"
    local _tsig_key="${2}"
    local _ttl="${3}"
    local _token="${4}"
    local _script=$( mktemp create-acme-challenge.XXXXXX )

    # Use the first authoritative DNS servers for the zone
    local _nameserver=$( dig +short "${_zone_name}" | head -1 )
    if -z "${_nameserver}"; then
	echo "Unable to find authoritative DNS servers for ${_zone_name}"
	exit 1
    fi

    cat > "${_script}" <<__EOF__
debug yes
server ${_nameserver}
zone ${_zone_name}
update delete _acme-challenge.${_zone_name} TXT ${_token}
send
__EOF__

    nsupdate -k "${_tsig_key}" -v "${_script}"
    rm -f "${_script}"
}

# Main entrypoint
function _main() {
    local _cmd="${1}"
    local _zone="${2}"
    local _tsig_key="${3}"
    local _ttl="${4}"
    local _token="${5}"

    if [ $# -ne 5 ]; then
	_usage
    fi

    case "${_cmd}" in
	create)
	    _add_acme_challenge_record "${_zone}" "${_tsig_key}" "${_ttl}" "${_token}"
	    ;;
	delete)
	    _delete_acme_challenge_record "${_zone}" "${_tsig_key}" "${_ttl}" "${_token}"
	    ;;
	*)
	    _usage
	    ;;
    esac
}

_main $*
