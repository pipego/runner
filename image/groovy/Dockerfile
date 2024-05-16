FROM jenkins/jenkinsfile-runner:latest

COPY plugins.yml /usr/share/jenkins/ref/plugins.yml

RUN cd /app/jenkins && jar -cvf jenkins.war *
RUN java -jar /app/bin/jenkins-plugin-manager.jar --war /app/jenkins/jenkins.war --plugin-file /usr/share/jenkins/ref/plugins.yml && rm /app/jenkins/jenkins.war
