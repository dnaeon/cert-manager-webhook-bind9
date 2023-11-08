#!/usr/bin/env bash
#
# A helper script which uses nsupdate(1) to handle ACME DNS-01
# challenges
#

set -e

# Just in case if it's not setup on the running environment
TMPDIR=${TMPDIR:-"/tmp"}

# Force DNS queries to be directed against this nameserver, instead of
# querying the zone for the NS records.
USE_NAMESERVER=${USE_NAMESERVER:-}

_SCRIPT_NAME="${0##*/}"

# Prints the usage of the sript
function _usage() {
    echo "${_SCRIPT_NAME} [create|delete] <zone> <fqdn> <tsig-key> <ttl> <token>"
    exit 64  # EX_USAGE
}

# Handles the ACME challenge by either creating or deleting the
# respective DNS TXT record
#
# $1: Operation (either create or delete)
# $2: Zone name
# $3: FQDN
# $4: Path to TSIG key
# $5: TTL
# $6: Token / Key
function _handle_acme_challenge() {
    local _op="${1}"
    local _zone_name="${2}"
    local _fqdn="${3}"
    local _tsig_key="${4}"
    local _ttl="${5}"
    local _token="${6}"

    # The operation we are about to perform
    local _operation=""
    local _op_add="update add ${_fqdn} ${_ttl} TXT ${_token}"
    local _op_delete="update delete ${_fqdn} TXT ${_token}"
    case "${_op}" in
	create)
	    _operation="${_op_add}"
	    ;;
	delete)
	    _operation="${_op_delete}"
	    ;;
	*)
	    _usage
	    ;;
    esac

    local _nameserver=""
    local _script=$( mktemp nsupdate-script.XXXXXX )

    # If $USE_NAMESERVER is specified forward queries to this
    # nameserver, otherwise run the queries against the authoritative
    # nameservers.
    if [ ! -z "${USE_NAMESERVER}" ]; then
	_nameserver="${USE_NAMESERVER}"
    else
	# Use the first authoritative DNS servers for the zone
	local _nameserver=$( dig +short -t ns "${_zone_name}" | head -1 )
    fi

    # We should have a nameserver in all cases
    if [ -z "${_nameserver}" ]; then
	echo "Unable to find authoritative DNS servers for ${_zone_name}"
	exit 1
    fi

    cat > "${_script}" <<__EOF__
debug yes
server ${_nameserver}
zone ${_zone_name}
${_operation}
send
__EOF__

    nsupdate -k "${_tsig_key}" -v "${_script}"
    rm -f "${_script}"
}

# Main entrypoint
function _main() {
    local _cmd="${1}"
    local _zone="${2}"
    local _fqdn="${3}"
    local _tsig_key="${4}"
    local _ttl="${5}"
    local _token="${6}"

    if [ $# -ne 6 ]; then
	_usage
    fi

    _handle_acme_challenge "${_cmd}" "${_zone}" "${_fqdn}" "${_tsig_key}" "${_ttl}" "${_token}"
}

_main $*
