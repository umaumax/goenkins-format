import java.text.SimpleDateFormat

pipeline {
agent none // comment!!!!!
// comment
/*
comment!!!!!!
*/
/**/
/***/
stages {
stage('publish-html') {
agent any
steps {
script {
def dateFormat = new SimpleDateFormat("yyyyMMddHHmmss") // comment
def current = dateFormat.format(new Date())
def baseDir = '/var/www/'

// WARN: a /*sample*/ = c
env.workDir=baseDir+'/'+current
env.htmlDir=baseDir+'/'+'html'
}
sh "mv _build/html ${env.workDir}"
sh "ln -sfn ${env.workDir} ${env.htmlDir}"
sh '''
        echo hello
        '''
sh """
        echo hello
        """
}
}
}
}

