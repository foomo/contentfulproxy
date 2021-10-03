#!/bin/bash

USER="foomo"
NAME="contentfulproxy"
URL="http://www.foomo.org"
DESCRIPTION="An experimental proxy for read access to contentful to save your API quota"
LICENSE="LGPL-3.0"

# get version
VERSION=`bin/contentfulproxy --version | sed 's/contentfulproxy //'`

# create temp dir
TEMP=`pwd`/pkg/tmp
mkdir -p $TEMP

package()
{
	OS=$1
	ARCH=$2
	TYPE=$3
	TARGET=$4

	# copy license file
	cp LICENSE $LICENSE

	# define source dir
	SOURCE=`pwd`/pkg/${TYPE}

	# create build folder
	BUILD=${TEMP}/${NAME}-${VERSION}
	#rsync -rv --exclude **/.git* --exclude /*.sh $SOURCE/ $BUILD/

	# build binary
	GOOS=$OS GOARCH=$ARCH go build -o $BUILD/usr/local/bin/${NAME}

	# create package
	fpm -s dir \
		-t $TYPE \
		--name $NAME \
		--maintainer $USER \
		--version $VERSION \
		--license $LICENSE \
		--description "${DESCRIPTION}" \
		--architecture $ARCH \
		--package $TEMP \
		--url "${URL}" \
		-C $BUILD \
		.

	# push
	package_cloud push $TARGET $TEMP/${NAME}_${VERSION}_${ARCH}.${TYPE}

	# cleanup
	rm -rf $TEMP
	rm $LICENSE
}

package linux amd64 deb foomo/contentfulproxy/ubuntu/precise
package linux amd64 deb foomo/contentfulproxy/ubuntu/trusty

#package linux amd64 rpm
