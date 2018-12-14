#!/usr/bin/env sh

JWT_SECRET=catchcatch
SECRET=6eCxyjW2fF5nlyJlfjPJ0aw4
SCOPE=https://www.googleapis.com/auth/userinfo.email
CLIENT_ID=107569463120-d4ar6krs9m465h8l4mdsr24d4b5r6tb6.apps.googleusercontent.com
PORT=8080

docker run  -p 8080:$PORT tarent/loginsrv \
    -jwt-secret $JWT_SECRET \
    -google "client_id=$CLIENT_ID,client_secret=$SECRET,scope=$SCOPE" \
    -success-url http://localhost:5000/web/admin \
    -cookie-domain localhost -cookie-name=X-CatchCatch-Auth \
