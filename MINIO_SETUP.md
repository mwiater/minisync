# MinIO Cluster Setup on Raspberry Pi 4

## Introduction

MinIO is a high-performance, distributed object storage system that is compatible with Amazon S3 APIs. This guide will help you set up a MinIO cluster on Raspberry Pi 4 devices running Ubuntu, with storage on USB drives. By the end of this guide, you'll have a functional, resilient storage solution suitable for small-scale projects or personal use.

## Prerequisites

1. **Ubuntu Installed** on each Raspberry Pi.
2. **Static IP Addresses** for each Raspberry Pi:
    - `192.168.0.81`
    - `192.168.0.82`
    - `192.168.0.83`
    - `192.168.0.84`
3. **USB Drives** connected and mounted on each Raspberry Pi.

## Step 1: Update and Install Necessary Packages

1. **Update Package Lists**:
    ```bash
    sudo apt update && sudo apt upgrade -y
    ```

2. **Install Dependencies**:
    ```bash
    sudo apt install -y wget unzip
    ```

## Step 2: Mount USB Drives

1. **Identify USB Drive**:
    ```bash
    sudo fdisk -l
    ```

    Example output:
    ```bash
    Disk /dev/sda: 29.32 GiB, 31482445824 bytes, 61489152 sectors
    Disk model: Cruzer Fit
    Units: sectors of 1 * 512 = 512 bytes
    Sector size (logical/physical): 512 bytes / 512 bytes
    I/O size (minimum/optimal): 512 bytes / 512 bytes
    Disklabel type: gpt
    Disk identifier: 2A93212D-ABD4-486B-B063-B021BCA2C9FB
    ```

2. **Create a Mount Point**:
    ```bash
    sudo mkdir -p /mnt/minio
    ```

3. **Mount the USB Drive**:
    Replace `/dev/sda1` with your actual USB drive identifier.
    ```bash
    sudo mount /dev/sda /mnt/minio
    ```

4. **OPTIONAL: Format the Drive if Necessary**:
    Replace `/dev/sda1` with your actual USB drive identifier.
    ```bash
    sudo umount /dev/sda
    sudo mkfs.ext4 /dev/sda
    sudo mkdir -p /mnt/minio
    sudo mount /dev/sda /mnt/minio
    ```

5. **Ensure USB Drive Mounts on Boot**:
    Add the following line to `/etc/fstab`:
    ```bash
    sudo nano /etc/fstab
    ```

    Add this line to the file:
    ```bash
    /dev/sda /mnt/minio ext4 defaults 0 0
    ```

## Step 3: Download and Install MinIO

1. **Download MinIO**:
    ```bash
    wget https://dl.min.io/server/minio/release/linux-arm64/minio
    ```

2. **Make MinIO Executable**:
    ```bash
    chmod +x minio
    ```

3. **Move MinIO to `/usr/local/bin/`**:
    ```bash
    sudo mv minio /usr/local/bin/
    ```

## Step 4: Start MinIO on All Nodes

1. **Start MinIO on Each Raspberry Pi**:
    ```bash
    minio server http://192.168.0.81/mnt/minio http://192.168.0.82/mnt/minio http://192.168.0.83/mnt/minio http://192.168.0.84/mnt/minio
    ```

2. **Create a Systemd Service** (Optional):
    Create a service file at `/etc/systemd/system/minio.service` with the following content:
    ```ini
     [Unit]
    Description=MinIO
    Documentation=https://docs.min.io
    Wants=network-online.target
    After=network-online.target

    [Service]
    User=root
    Group=root
    ExecStart=/usr/local/bin/minio server http://192.168.0.81/mnt/minio/data http://192.168.0.82/mnt/minio/data http://192.168.0.83/mnt/minio/data http://192.168.0.84/mnt/minio/data
    Restart=always
    Environment=<MINIO_ACCESS_KEY> 
    Environment=<MINIO_SECRET_KEY>
    EnvironmentFile=-/etc/default/minio
    LimitNOFILE=65536

    [Install]
    WantedBy=multi-user.target
    ```

3. **Enable and Start the Service**:
    ```bash
    sudo systemctl daemon-reload
    sudo systemctl enable minio
    sudo systemctl start minio
    ```

4. Install the MinIO Client

    ```bash
    wget https://dl.min.io/client/mc/release/linux-arm64/mc
    sudo chmod +x mc
    sudo mv mc /usr/local/bin/
    ```

    Configure the MinIO Client (mc):

    ```bash
    mc alias set minisync http://192.168.0.81:9000 <MINIO_ACCESS_KEY> <MINIO_SECRET_KEY>
    ```

    ```
    Added `minisync` successfully.
    ```

    Add bucket for minisync go application:
    ```
    mc mb minisync/minisync
    ```

## Step 5: Access the MinIO Web Interface

- Open your browser and navigate to `http://192.168.0.81:9000`.
- Log in using your MinIO access key and secret key.

## Bucket Management

### Creating Buckets
```bash
mc mb minisync/mybucket
```

### Listing Buckets
```bash
mc ls minisync
```

### Removing Buckets
```bash
mc rb minisync/mybucket
```

## Object Management

### Uploading Objects
```bash
mc cp /path/to/file minisync/mybucket
```

### Downloading Objects
```bash
mc cp minisync/mybucket/file /path/to/destination
```

### Managing Objects

1. **Remove an Object**:
    ```bash
    mc rm minisync/mybucket/file
    ```

2. **Remove All Objects from a Bucket**:
    ```bash
    mc rm --recursive --force minisync/mybucket
    ```

3. **Move an Object**:
    ```bash
    mc mv minisync/mybucket/sourcefile minisync/mybucket/destinationfile
    ```

4. **Copy an Object**:
    ```bash
    mc cp minisync/mybucket/sourcefile minisync/mybucket/destinationfile
    ```

## Managing Policies and Users

1. **Create a New User**:
    ```bash
    mc admin user add minisync newuser newuserpassword
    ```

2. **Disable a User**:
    ```bash
    mc admin user disable minisync newuser
    ```

3. **Enable a User**:
    ```bash
    mc admin user enable minisync newuser
    ```

4. **Remove a User**:
    ```bash
    mc admin user remove minisync newuser
    ```

5. **Add a Policy to a User**:
    ```bash
    mc admin policy set minisync readwrite user=newuser
    ```

6. **List All Policies**:
    ```bash
    mc admin policy list minisync
    ```

7. **Create a Custom Policy**:
    ```bash
    mc admin policy add minisync custompolicy /path/to/policy.json
    ```

## Monitoring and Diagnostics

1. **Check Server Status**:
    ```bash
    mc admin info minisync
    ```

2. **Monitor Server Logs**:
    ```bash
    mc admin log minisync
    ```

3. **Check Server Health**:
    ```bash
    mc admin health minisync
    ```

4. **View Bucket Usage**:
    ```bash
    mc du minisync/mybucket
    ```

### Example Policy File (`policy.json`)

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "s3:GetObject",
                "s3:PutObject"
            ],
            "Resource": [
                "arn:aws:s3:::mybucket/*"
            ]
        }
    ]
}
```

## Troubleshooting

### Common Issues

1. **MinIO Server Fails to Start**:
   - Ensure all Raspberry Pi devices have consistent system time and are reachable over the network.
   - Check that the USB drives are correctly mounted and accessible.

2. **Unable to Access MinIO Web Interface**:
   - Verify that the correct port is open and that the MinIO server is running.

## Summary

You have successfully set up a MinIO cluster on your Raspberry Pi devices. Your cluster is now ready to store and manage your data with high availability and resilience. Next, consider setting up regular backups, monitoring the health of your cluster, and exploring MinIO's advanced features.