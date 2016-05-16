#!/bin/bash -e

function contains() {
    local n=$#
    local value=${!n}
    for ((i=1;i < $#;i++)) {
        if [ "${!i}" == "${value}" ]; then
            echo "y"
            return 0
        fi
    }
    echo "n"
    return 1
}

function tempfile() {
    if [ "$(pick_os)" == "osx" ]; then
        mktemp -t secretshare
    elif [ "$(pick_os)" == "linux" ]; then
        mktemp secretshare.XXXXXXXX
    fi
}

function missing_aws_config() {
	echo "  You have no credentials in $HOME/.aws/config. Please enter the AWS credentials"
	echo "  you want to use to set up the AWS user and S3 bucket that secretshare needs."
	echo
	echo "  These AWS credentials must have admin privileges. They will not be remembered"
	echo "  by secretshare after this initial setup process."
	echo
	
	echo -n "  AWS access key ID: "
	read aws_access_key_id
	while ! echo "${aws_access_key_id}" | egrep "^AKI" >/dev/null; do
		echo "  AWS access key ID should begin with 'AKI'. Try again."
		echo
		echo -n "  AWS access key ID: "
		read aws_access_key_id
	done

	echo -n "  AWS secret access key: "
	read aws_secret_access_key
	while [ -z "${aws_secret_access_key}" ]; do
		echo "  That's not a thing. Try again."
		echo
		echo -n "  AWS secret access key: "
		read aws_secret_access_key
	done

	echo -n "  AWS region: "
	read aws_region
	while ! echo "${aws_region}" | egrep -- '.+-.+-[0-9]$' >/dev/null; do
		echo "  Enter an AWS region name, like 'us-west-2' or 'ap-northeast-1'. Try again."
		echo
		echo -n "  AWS region: "
		read aws_region
	done

	mkdir -p "${HOME}/.aws"
	chmod 700 "${HOME}/.aws"
	cat > "${HOME}/.aws/config" <<EOF
[default]
aws_access_key_id = ${aws_access_key_id}
aws_secret_access_key = ${aws_secret_access_key}
region = ${aws_region}
EOF
}

function pick_bind_ip() {
	echo >&2 -n "  Binding IP (default: 127.0.0.1): "
	read bind_ip
	if [ -z "${bind_ip}" ]; then
		bind_ip=127.0.0.1
	fi
	echo "${bind_ip}"
}

function pick_bind_port() {
	echo >&2 -n "  Binding port (default: 8080): "
	read bind_port
	if [ -z "${bind_port}" ]; then
		bind_port=8080
	fi
	echo "${bind_port}"
}

function pick_server_endpoint() {
	default="${1}"
	echo >&2 -n "  The URL at which secretshare-server will be hosted (default: ${default}): "
	read server_endpoint
	if [ -z "${server_endpoint}" ]; then
		server_endpoint="${default}"
	fi
	echo "${server_endpoint}"
}

function pick_aws_profile() {
	echo >&2 "  Pick an AWS profile to use for initial AWS user and S3 bucket setup."
	echo >&2
	echo >&2 "  These AWS credentials must have admin privileges. They will not be remembered"
	echo >&2 "  by secretshare after this initial setup process."
	echo >&2

	echo >&2 "  Here are the available profiles in ${HOME}/.aws/config:"
	echo >&2
	profiles=( $(egrep '^\[' "${HOME}/.aws/config" | sed -Ee 's/\[(profile )?(.*)]$/\2/') )
	for prof in "${profiles[@]}"; do
		echo >&2 "    ${prof}"
	done
	echo >&2

	echo >&2 -n "  AWS profile name: "
	read aws_profile
	while [ $(contains "${profiles[@]}" "${aws_profile}") != "y" ]; do
		echo >&2 "  Enter one of the AWS profile names above. Try again."
		echo >&2
		echo >&2 -n "  AWS profile name: "
		read aws_profile
	done
	echo "${aws_profile}"
}

function bucket_writable() {
	aws_profile="${1}"
	bucket="${2}"
	test_contents="${RANDOM}"
	infile="$(tempfile)"
	outfile="$(tempfile)"
	echo "${test_contents}" > "${infile}"
	if ! aws --profile "${aws_profile}" s3 cp --quiet "${infile}" "s3://${bucket}/test.txt"; then
		echo "n"
		return
	fi
	if ! aws --profile "${aws_profile}" s3 cp --quiet "s3://${bucket}/test.txt" "${outfile}"; then
		echo "n"
		return
	fi
	if ! diff >&2 "${infile}" "${outfile}"; then
		echo "n"
		return
	fi
	if ! aws --profile "${aws_profile}" s3 rm --quiet "s3://${bucket}/test.txt"; then
		echo "n"
		return
	fi
	rm -f "${infile}" "${outfile}"
	echo "y"
}

function pick_bucket() {
	aws_profile="${1}"
	echo >&2 -n "  S3 bucket name: "
	read bucket
	while [ "$(bucket_writable "${profile}" "${bucket}")" != "y" ]; do
		echo >&2 "  The bucket '${bucket}' doesn't exist in the specified region, or the"
		echo >&2 "  given credentials can't access it. Try again."
		echo >&2
		echo >&2 -n "  S3 bucket name: "
		read bucket
	done
	echo "${bucket}"
}

function create_bucket() {
	aws_profile="${1}"
	while [ -z "${bucket}" ]; do
		echo >&2 -n "  New S3 bucket name: "
		read bucket
	done
	aws --profile "${aws_profile}" s3api create-bucket --bucket "${bucket}" >/dev/null
    aws --profile "${aws_profile}" s3api put-bucket-lifecycle --bucket "${bucket}" --lifecycle-configuration '{"Rules":[{"Prefix":"/","Status":"Enabled","Expiration":{"Days":1}}]}' >/dev/null

	echo "${bucket}"
}

function create_or_pick_bucket() {
	aws_profile="${1}"
	while [ "${yn}" != "y" ] && [ "${yn}" != "n" ]; do
		echo >&2 -n "  Is there already an S3 bucket with which you want to use secretshare? (y/n) "
		read yn
	done
	echo >&2
	if [ "${yn}" == "y" ]; then
		echo "$(pick_bucket "${aws_profile}")"
		return
	fi
	echo "$(create_bucket "${aws_profile}")"
}

function gen_secretshare_key() {
	echo -n $(LC_CTYPE=C tr -dc A-Za-z0-9 < /dev/urandom | fold -w ${1:-64} | head -n 1)
}

function pick_iam_user() {
	echo >&2 -n "  AWS access key ID for secretshare user: "
	read aws_access_key_id
	while ! echo "${aws_access_key_id}" | egrep "^AKI" >/dev/null; do
		echo >&2 "  AWS access key ID should begin with 'AKI'. Try again."
		echo >&2
		echo >&2 -n "  AWS access key ID: "
		read aws_access_key_id
	done

	echo >&2 -n "  AWS secret access key for secretshare user: "
	read aws_secret_access_key
	while [ -z "${aws_secret_access_key}" ]; do
		echo >&2 "  That's not a thing. Try again."
		echo >&2
		echo >&2 -n "  AWS secret access key: "
		read aws_secret_access_key
	done

	echo "${aws_access_key_id}:${aws_secret_access_key}"
}

function create_iam_user() {
	aws_profile="${1}"
	bucket="${2}"
	echo >&2 -n "  Username for IAM user to create [default: secretshare]: "
	read username
	if [ -z "${username}" ]; then
		username=secretshare
	fi
	aws --profile "${aws_profile}" iam create-user --user-name "${username}" >/dev/null
	policy_name="secretshare-${RANDOM}-${RANDOM}"
	policy_file=$(tempfile)
	cat >"${policy_file}" <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "s3:DeleteObject",
                "s3:DeleteObjectVersion",
                "s3:GetObject",
                "s3:GetObjectAcl",
                "s3:GetObjectTorrent",
                "s3:GetObjectVersion",
                "s3:GetObjectVersionAcl",
                "s3:GetObjectVersionTorrent",
                "s3:ListBucket",
                "s3:PutObject",
                "s3:PutObjectAcl",
                "s3:PutObjectVersionAcl"
            ],
            "Resource": [
                "arn:aws:s3:::${bucket}/*"
            ]
        }
    ]
}
EOF
	policy_arn=$(aws --profile "${aws_profile}" iam create-policy --policy-name "${policy_name}" --policy-document "file://${policy_file}" | grep '"Arn"' | cut -d\" -f4)
	rm -f "${policy_file}"
	aws --profile "${aws_profile}" iam attach-user-policy --policy-arn "${policy_arn}" --user-name "${username}" >/dev/null
	access_key_output=$(aws --profile "${aws_profile}" --output text iam create-access-key --user-name "${username}")
	awk '{print $2":"$4}' <<<"${access_key_output}"
}

function create_or_pick_iam_user() {
	aws_profile="${1}"
	bucket="${2}"
	while [ "${yn}" != "y" ] && [ "${yn}" != "n" ]; do
		echo >&2 "  secretshare needs an IAM user with enough privileges to manipulate"
		echo >&2 "  the secretshare bucket. If you don't have one in mind, we can create"
		echo >&2 "  one for you and give it a minimal set of privileges."
		echo >&2 
		echo >&2 -n "  Is there already an IAM user you want to use for secretshare? (y/n) "
		read yn
	done
	echo >&2
	if [ "${yn}" == "y" ]; then
		echo "$(pick_iam_user "${aws_profile}")"
		return
	fi
	echo "$(create_iam_user "${aws_profile}" "${bucket}")"
}

function pick_os() {
	if [ "$(uname -s)" == "Darwin" ]; then
		echo "osx"
		return
	fi
	echo "linux"
}

function pick_arch() {
	go version | cut -d/ -f2
}

if ! [ -d "${GOPATH}/src" ]; then
	echo "You must export \$GOPATH before running this script, and it must point to"
	echo "an existing go directory tree."
	exit 1
fi
if [ "$(pwd)" != "${GOPATH}/src/github.com/waucka/secretshare" ]; then
	echo "This script must be run from ${GOPATH}/src/github.com/waucka/secretshare"
	exit 1
fi
if ! [ -x "$(which aws)" ]; then
	echo "Install the AWS command-line interface with 'pip install awscli' before"
	echo "running this setup script."
	exit 1
fi

echo "We will now set you up to hack on secretshare."
echo

step=0


step=$((step+1))
echo "${step} Create build output directories"
echo
mkdir -p build/{linux,osx,win}-amd64


step=$((step+1))
echo "${step} Choose the binding address and port for the secretshare server"
echo
echo "  You can always change this later by editing secretshare-server.json. To listen on"
echo "  all IP addresses, choose 0.0.0.0."
echo
bind_ip=$(pick_bind_ip)
bind_port=$(pick_bind_port)
server_endpoint=$(pick_server_endpoint "http://${bind_ip}:${bind_port}")
echo


step=$((step+1))
echo "${step} Set up AWS credentials for initial bucket creation and permissions management"
echo
if ! [ -e "${HOME}/.aws/config" ]; then
	missing_aws_config
	profile="default"
elif ! [ -r "${HOME}/.aws/config" ]; then
	echo "Cannot read AWS credentials from '${HOME}/.aws/config'"
	exit 1
else
	profile=$(pick_aws_profile)
fi
region=$(aws --profile "${profile}" configure get region)


step=$((step+1))
echo
echo "${step} Establish S3 bucket"
echo
bucket=$(create_or_pick_bucket "${profile}")


step=$((step+1))
echo
echo "${step} Populate test_env"
echo
current_os=$(pick_os)
current_arch=$(pick_arch)
cat >test_env <<EOF
export CURRENT_OS=${current_os}
export CURRENT_ARCH=${current_arch}
export TEST_BUCKET_REGION=${region}
export TEST_BUCKET=${bucket}
EOF


step=$((step+1))
echo
echo "${step} Set up AWS credentials for secretshare user"
echo
secretshare_creds=$(create_or_pick_iam_user "${profile}" "${bucket}")
aws_keyid=$(cut -d: -f1 <<<"${secretshare_creds}")
aws_secret=$(cut -d: -f2 <<<"${secretshare_creds}")
cat >"${HOME}/.aws/credentials.secretshare" <<EOF
[default]
aws_access_key_id = ${aws_keyid}
aws_secret_access_key = ${aws_secret}
region = ${region}
EOF
if [ -e "${HOME}/.aws/credentials" ] && ! [ -e "${HOME}/.aws/credentials.normal" ]; then
	cp "${HOME}/.aws/credentials" "${HOME}/.aws/credentials.normal"
else
	# If this is empty, "credmgr off" will just delete the credentials file
	touch "${HOME}/.aws/credentials.normal"
fi


step=$((step+1))
echo
echo "${step} Populate client and server config files"
echo
secretshare_key=$(gen_secretshare_key)
sed -e "s!http://localhost:8080!${server_endpoint}!; s/us-west-1/${region}/; s/secretshare/${bucket}/; s/THISISABADKEY/${secretshare_key}/" vars.json.example > vars.json
sed -e "s/0.0.0.0/${bind_ip}/; s/8080/${bind_port}/; s/THISISABADKEY/${secretshare_key}/; s/%AWS_ACCESS_KEY_ID%/${aws_keyid}/; s/%AWS_SECRET_ACCESS_KEY%/${aws_secret}/" secretshare-server.json.example > secretshare-server.json


echo
echo "Done!"
echo
echo "_____________ If you're hacking _____________"
echo
echo "The credentials with which the server will run have been written to"
echo "~/.aws/credentials.secretshare. To start hacking on secretshare, you'll:"
echo "need these credentials in ~/.aws/credentials. But that may interfere"
echo "with the operation of libraries that read the credentials file (e.g."
echo "boto and aws-sdk-go)."
echo 
echo "So we've provided a convenience script for swapping out credentials as"
echo "necessary. To start working on secretshare:"
echo
echo "./credmgr on"
echo
echo "And when you're done:"
echo
echo "./credmgr off"
echo
echo "Go ahead. Try \"./credmgr on && source test_env && make test\"."
echo
echo "_____________ If you're installing for real _____________"
echo
echo "Your binaries are in the build directory. Copy secretshare-server to"
echo "/usr/local/bin on the target server. Copy secretshare-server.json to"
echo "/etc on the target server. Then start it."
echo
echo "Once the server is running, send out the various secretshare binaries"
echo "(for different operating systems) to your users, along with the"
echo "following command to initialize their setup:"
echo
echo "secretshare config --endpoint '${server_endpoint}' --bucket-region '${region}' --bucket '${bucket}' --auth-key '${secretshare_key}'"
echo
echo "After they run that command, they should be good to go."
echo
echo "Enjoy! And if you find any bugs or want to suggest any improvements,"
echo "hit us up at https://github.com/secretshare."
