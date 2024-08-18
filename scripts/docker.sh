#!/bin/bash
default_helper() {
    if [ $1 -eq 1 ]; then
        echo "${2} is not a valid argument, please follow types below"
    fi

    echo "
    kubefs docker - create and test docker images for created resources to be deployed onto the clusters

    kubefs docker build all - build for all components
    kubefs docker build <name> - build for singular component
    kubefs docker exec all - run all components from created docker images
    kubefs docker exec <name> - run singular component from created docker image
    "
}

declare -a containers

build_unique(){
    NAME=$1
    SCRIPT_DIR=$2
    CURRENT_DIR=`pwd`

    if [ -z $NAME ]; then
        default_helper 1 $NAME
        return 1
    fi

    if [ ! -f "$CURRENT_DIR/$NAME/scaffold.kubefs" ]; then
        default_helper 1 $NAME
        return 1
    fi

    eval "$(parse_scaffold "$NAME")"

    case "${scaffold_data["type"]}" in
        "api")
            sed -e "s/{{PORT}}/${scaffold_data["port"]}/" \
                -e "s/{{NAME}}/${scaffold_data["name"]}/" \
                "$SCRIPT_DIR/scripts/templates/template-api-dockerfile.conf" > "$CURRENT_DIR/$NAME/Dockerfile";;
        "frontend")
            sed -e "s/{{PORT}}/${scaffold_data["port"]}/" \
                "$SCRIPT_DIR/scripts/templates/nginx.conf" > "$CURRENT_DIR/$NAME/nginx.conf"
            sed -e "s/{{PORT}}/${scaffold_data["port"]}/" \
                "$SCRIPT_DIR/scripts/templates/template-frontend-dockerfile.conf" > "$CURRENT_DIR/$NAME/Dockerfile";;
        *) default_helper 1 "${scaffold_data["type"]}";;
    esac

    # build docker image
    cd $CURRENT_DIR/$NAME && docker buildx build -t $NAME .

    echo "$NAME component build successfuly, run using 'kubefs docker exec'"

    if [ -z "${scaffold_data["docker-run"]}" ]; then
        echo "docker-run=docker run -p ${scaffold_data["port"]}:${scaffold_data["port"]} --name $NAME-container $NAME" >> $CURRENT_DIR/$NAME/scaffold.kubefs
    fi

    return 0
}

build(){
    SCRIPT_DIR=$1
    name=$2

    if [ -z $name ]; then
        default_helper 0
        return 1
    fi

    case $type in
        "all") build_all $SCRIPT_DIR;;
        "--help") default_helper 0;;
        *) build_unique $name $SCRIPT_DIR;;
    esac
}

execute_unique(){
    NAME=$1
    SCRIPT_DIR=$2
    CURRENT_DIR=`pwd`

    if [ -z $NAME ]; then
        default_helper 1 $NAME
        return 1
    fi

    if [ ! -f "$CURRENT_DIR/$NAME/scaffold.kubefs" ]; then
        default_helper 1 $NAME
        return 1
    fi

    eval "$(parse_scaffold "$NAME")"

    if [ -z "${scaffold_data["docker-run"]}" ]; then
        echo "Docker image not built for $NAME, please build using 'kubefs docker build'. "
        return 1
    fi

    echo "Running $NAME component on port ${scaffold_data["port"]} using docker image $NAME..."
    ${scaffold_data["docker-run"]} > /dev/null 2>&1

    containers+=($NAME-container)

    exit_flag=0
    while [ "$exit_flag" -eq "0" ]; do
        sleep 1
    done
    

    return 0
}

execute(){
    SCRIPT_DIR=$1
    name=$2

    if [ -z $name ]; then
        default_helper 0
        return 1
    fi

    case $type in
        "all") execute_all $SCRIPT_DIR;;
        "--help") default_helper 0;;
        *) execute_unique $name $SCRIPT_DIR;;
    esac
}

cleanup(){
    for container in "${containers[@]}"; do
        echo ""
        echo "Stopping $container..."
        docker stop $container > /dev/null 2>&1
        # docker rm $container > /dev/null 2>&1
    done
    containers=()
    exit_flag=1
    exit 0
}

trap cleanup SIGINT

main(){
    SCRIPT_DIR="$(dirname "$(readlink -f "$0")")"

    if [ -z $1 ]; then
        default_helper 0
        return 1
    fi

    # source helper functions 
    source $SCRIPT_DIR/scripts/helper.sh
    validate_project

    type=$1
    shift
    case $type in
        "build") build $SCRIPT_DIR $@;;
        "exec") execute $SCRIPT_DIR $@;;
        "--help") default_helper 0;;
        *) default_helper 1 $type;;
    esac
}

main $@
exit 0


