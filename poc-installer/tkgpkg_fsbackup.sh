##############################################################################
# This is a simple backup script.                                            #
# It accepts a .tar (not a .gz or .bz compressed) or .deb as a first argument#
# and creates a backup of the current HOST file system based on its content. #
# Its sole purpose is to create a backup prior to extracting the package.    #
# NOTE: the package must be a file system structure                          #
##############################################################################

#!/bin/bash

#.DATA SECTION (as we say in assembler jargon)
#here we just set some global consts
##################
readonly ARG1="$1"
readonly BAK_DIR="/tmp/byoh/bak"
readonly RESTORE_DIR="fs"
readonly WORK_DIR="/tmp/byoh"
readonly PAYLOAD_DIR="./payload"

#HELPER FUNCTIONS
##################

# accepts: file or dir absolute path
# returns: 0 = does not exists; 1 = it's a file; 2 = it's a dir
fileOrDirExists() 
{
    fsItem=$1
    if [ -d "$fsItem" ]; then
        return 2 #dir exists
    else
        if [ -f "$fsItem" ]; then
            return 1 #file exists
        else
            return 0 #fs object does not exists
        fi
    fi
}

#create dir in the backup if required
backupMkdirIfNotExists() 
{
    dir=$1
    fileOrDirExists "$BAK_DIR/$RESTORE_DIR/$dir"
    if [ $? -eq 0 ]; then
        mkdir "$BAK_DIR/$RESTORE_DIR/$dir"
    fi 
}

#backup a file to the appropriate backup dir
backupFile()
{
    fileItem=$1
    cp "$fileItem" "$BAK_DIR/$RESTORE_DIR$fileItem"

    #update the restore.list only if needed
    if grep -Fxq "$fileItem" $BAK_DIR/restore.list 2> /dev/null
    then
        : # don't add the entry if it already exists
    else
        echo "$fileItem" >> $BAK_DIR/restore.list
    fi
}

backupAddRemoveItemEntry()
{
    fileItem=$1
    #update the remove.list only if needed
    if grep -Fxq "$fileItem" $BAK_DIR/remove.list 2> /dev/null
    then
        : # don't add it if found
    else
        echo "$fileItem" >> $BAK_DIR/remove.list
    fi
}

isDebOrTar()
{
    fileItem=$1
    filename=$(basename -- "$fileItem")
    extension="${fileItem##*.}"

    if [ "$extension" = "tar" ]; then
        return 1
    elif [ "$extension" = "deb" ]; then
        return 2
    else
        return 3
    fi
}

recursiveExtractLayers()
{
    if [ ! -d "$WORK_DIR/out" ]
    then
        mkdir "$WORK_DIR/out"
    fi

    echo "Extracting layer.tar archives to $WORK_DIR/out..."

    for layerTar in $(find $WORK_DIR -name "layer.tar")
    do
        tar -C "$WORK_DIR/out" -xf "$layerTar"
    done

    echo "Done."
}

recursiveListLayers()
{
    for layerTar in $(find $WORK_DIR -name "layer.tar")
    do
        tar -tf $layerTar >> $BAK_DIR/$archiveList.list
    done
}

isOciCompliantTar()
{
    fileObject=$1
    tarContent=$(tar -tf "$fileObject")

    if [ -z "${tarContent##*layer.tar*}" ] ;then
        return 1
    fi

    return 0
}

extractOciCompliantTar()
{
    fileName=$(basename -- $1)
    mkdir "$WORK_DIR/$fileName" && tar -C "$WORK_DIR/$fileName" -xvf "$1"
    recursiveListLayers
    rm -rf "$WORK_DIR/$fileName"
}

bakTar()
{
    isOciCompliantTar $1

    if [ $? -eq 1 ]; then #if OCI compliant archive (containing layer.tar archives)
        extractOciCompliantTar $1
    else #else if just plain file structure containing archive
        saveNewFsObjectsToList $1
    fi
}

bakDeb()
{
    dpkg -c $1 > $BAK_DIR/$archiveList.list
}

saveNewFsObjectsToList()
{
    fileContentList=$(tar -tf $1)

    echo "$fileContentList" |
    while IFS=/ read -r junk line; 
    do
        fileOrDirExists $line

        if [ $? -eq 0 ]; then
            echo $line >> $BAK_DIR/$archiveList.list;
        fi
    done
}

#ENTRY POINT
##################

#check for the command line argument (must be a tar or deb file path)
if [ -z $ARG1 ]; then
    echo "Pass the .tar or .deb path as fist argument"
    exit 0
else 
    echo "*** Processing: $ARG1"
fi

#creating the backup dir
if [ -d "$BAK_DIR/$RESTORE_DIR" ]; then
    : 
else
    mkdir -p $BAK_DIR/$RESTORE_DIR
fi

#getting the content of the package
if [ -f "$ARG1" ]; then
    archiveList="$(basename -- $ARG1)"

    isDebOrTar "$ARG1"
    debOrTar=$?

    if [ $debOrTar -eq 1 ]; then #tar
        bakTar $ARG1
        sed -i 's/.*\.\///' $BAK_DIR/$archiveList.list
    elif [ $debOrTar -eq 2 ]; then #deb
        # at this point do nothing, however keep it for a while
        # if needed, use it for deb file structure backup
        # also put the "sed -i" command above (see tar)
        # outside the current if-then-else statement

        : #bakDeb $ARG1
    else
        echo "The extension is not .deb nor .tar - exiting."
        exit 0
    fi

    
else 
    echo "$ARG1 does not exist, exiting..."
    exit 0
fi

#process the file structure of the package
#and create backup directories reflecting the file system
#as well as backing up any files that are also present in the package
#(meaning that they are about to be replaced after extraction)
while IFS= read -r line; do
    fsItem=/$line

    fileOrDirExists "$fsItem"
    dest_fileOrDirExists=$?

    if [ $dest_fileOrDirExists -eq 2 ]; then
        backupMkdirIfNotExists "$fsItem"
    else
        if [ $dest_fileOrDirExists -eq 1 ]; then
            backupFile "$fsItem"
        else 
            backupAddRemoveItemEntry "$fsItem"
        fi
    fi
done < $BAK_DIR/$archiveList.list

echo "Backup complete."