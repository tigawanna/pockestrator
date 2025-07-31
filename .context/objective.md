# pockestrator

This is supposed to be a project that manges pocketbase instances

When installed on a linux box it should ensure caddy is installed and configuered properly and should mimick bashscript below

```sh
#!/usr/bin/env bash
#  for oracle ampere
project_name="moots"
port="8094"
version="0.28.4"
pocketbase_url="https://github.com/pocketbase/pocketbase/releases/download/v${version}/pocketbase_${version}_linux_amd64.zip"

echo "========= downloading pocketbase version ${version} ======="
wget -q "$pocketbase_url"
echo "========= unzipping pocketbase version ${version} ======="

sudo apt install zip -y
sudo mkdir -p /home/ubuntu/${project_name}

sudo unzip -q pocketbase_${version}_linux_amd64.zip -d /home/ubuntu/${project_name}

sudo chmod +x /home/ubuntu/${project_name}/pocketbase
echo "========= pocketbase version ${version} has been downloaded and unzipped into /home/ubuntu/${project_name} successfully! ======="

sudo rm -rf pocketbase_${version}_linux_amd64.zip

echo "========= setting up a systemd service ======= "
# setup a systemd service service
sudo touch /lib/systemd/system/${project_name}-pocketbase.service
echo "
[Unit]
Description = ${project_name} pocketbase

[Service]
Type           = simple
User           = root
Group          = root
LimitNOFILE    = 4096
Restart        = always
RestartSec     = 5s
StandardOutput   = append:/home/ubuntu/${project_name}/errors.log
StandardError    = append:/home/ubuntu/${project_name}/errors.log
WorkingDirectory = /home/ubuntu/${project_name}/
ExecStart      = /home/ubuntu/${project_name}/pocketbase serve --http="127.0.0.1:${port}"

[Install]
WantedBy = multi-user.target
" | sudo tee /lib/systemd/system/${project_name}-pocketbase.service



sudo systemctl daemon-reload
sudo systemctl enable ${project_name}-pocketbase.service
sudo systemctl start ${project_name}-pocketbase

echo "========= creating default superuser ======="
# Wait a moment for the service to fully start
sleep 3
# Create default superuser
cd /home/ubuntu/${project_name}
sudo ./pocketbase superuser upsert denniskinuthiaw@gmail.com denniskinuthiaw@gmail.com

echo "========= adding caddy configuration ======="
# Add subdomain configuration to Caddyfile
caddy_config="
${project_name}.tigawanna.vip {
    request_body {
        max_size 10MB
    }
    reverse_proxy 127.0.0.1:${port} {
        transport http {
            read_timeout 360s
        }
        # Add these headers to forward client IP
        header_up X-Forwarded-For {remote_host}
        header_up X-Real-IP {remote_host}
    }
}
"

# Check if Caddyfile exists and add configuration
if [ -f "/etc/caddy/Caddyfile" ]; then
    echo "Adding ${project_name} subdomain configuration to Caddyfile..."
    echo "$caddy_config" | sudo tee -a /etc/caddy/Caddyfile
    echo "Reloading Caddy configuration..."
    sudo systemctl reload caddy
else
    echo "Warning: /etc/caddy/Caddyfile not found. Please add the following configuration manually:"
    echo "$caddy_config"
fi

echo "========= setup complete! ======="
echo "Project: ${project_name}"
echo "Port: ${port}"
echo "Subdomain: ${project_name}.tigawanna.vip"
echo "Service: ${project_name}-pocketbase.service"
```


you should break all the steps into go packages organized tidily in this priject to hadle a little bit of the above to tie together into a prject witha dashbord that allows you to list you regisred "services " which also double as system d services mapped to caddy config

The inputs should be project name , optional pocketbase version ( defaults to the latest version available)  and port ( defaults to the 8091 or the last part used + 1 if the last service was registerd wit 8091 the next should be 8091 , if the user provieded port or name exists it should throw an error on te dashbord that's picking it  )

we should have a dashboard that allows one to view the list of regisered projects and every project when listed should do a query to check if the config reqired for it to exst ( caddy and sysytemd ) are still correct , then a squigle should be shown forthe user to act upon , This should be a react app with tailwind + shadcn for the component librray and tanstack query for the data fetching , if any global state is required use zustand . as we're extending pocletbase most ofthe intrecations will pobably happen through the pocketbase api and the react app will be a client to that api wit the pocketbase sent
it should also use tantsck router for routing , but most importantly once the project is built as html css and js the resources should be embeddedinto the go binary dso it can be distributed as a single binary see #fetch : https://bindplane.com/blog/embed-react-in-golang

the detailed view of every regieted project should query the values saved to a pocketbase collection to ensure they math the actula values on te sytem e
g schek if the port regieted ipocket base matches whet we have in the caddy config and the port the ecex command in it's accompanying systemd config

we will be usisng pocketbase as a framework and add these steps 
ideally creating a new "service"

Intro #fetch: https://pocketbase.io/docs/go-overview/
Schedule job #fetch: https://pocketbase.io/docs/go-jobs-scheduling/
Sending emails #fetch: https://pocketbase.io/docs/go-sending-emails/

Record event hooks
#fetch: https://pocketbase.io/docs/go-event-hooks/#onrecordcreate
#fetch: https://pocketbase.io/docs/go-event-hooks/#onrecordcreateexecute
#fetch: https://pocketbase.io/docs/go-event-hooks/#onrecordaftercreatesuccess
#fetch: https://pocketbase.io/docs/go-event-hooks/#onrecordaftercreatesuccess

record model event hooks
#fetch: https://pocketbase.io/docs/go-records/

collection model hooks
#fetch: https://pocketbase.io/docs/go-collections/

database hooks and custom queries
#fetch: https://pocketbase.io/docs/go-database/

the collection services and creatinga new row should
 - create a new systemd service file
 - add a new caddy config to the Caddyfile
 - create a new pocketbase service with the provided name and port
- - the row should then be created as usual



