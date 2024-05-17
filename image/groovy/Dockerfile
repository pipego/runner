FROM mcr.microsoft.com/openjdk/jdk:17-ubuntu

USER root
RUN apt update && \
    apt install -y curl unzip && \
    mkdir /{jenkins,workspace}

USER root
WORKDIR /jenkins
COPY plugins.yml .
RUN curl -L https://github.com/jenkinsci/jenkins/releases/download/jenkins-2.452.1/jenkins.war -o jenkins.war && \
    unzip jenkins.war -d jenkins-war-files
RUN curl -L https://github.com/jenkinsci/jenkinsfile-runner/releases/download/1.0-beta-32/jenkinsfile-runner-1.0-beta-32.zip -o jenkinsfile-runner.zip && \
    mkdir jenkinsfile-runner && \
    unzip jenkinsfile-runner.zip -d jenkinsfile-runner && \
    chmod +x jenkinsfile-runner/bin/*
RUN curl -L https://github.com/jenkinsci/plugin-installation-manager-tool/releases/download/2.13.0/jenkins-plugin-manager-2.13.0.jar -o jenkins-plugin-manager.jar && \
    java -jar jenkins-plugin-manager.jar --war jenkins.war --plugin-download-directory plugins --plugin-file plugins.yml
RUN rm -rf *.jar *.war *.yml *.zip

USER root
WORKDIR /jenkins
ENV PATH=/jenkins/jenkinsfile-runner/bin:$PATH
ENTRYPOINT ["jenkinsfile-runner", "-ns", "-w", "jenkins-war-files", "-p", "plugins", "--runWorkspace", "workspace", "-f"]
CMD ["/workspace/jenkinsfile"]
