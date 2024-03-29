list =[
  'sample1',
  'sample2'
]

import groovy.json.*

def abc

parameters {
  booleanParam(defaultValue: true, description: '', name: 'flag')
  string(defaultValue: '', description: '', name: 'SOME_STRING')
}

pipeline {
  parameters {
    string(
      defaultValue: '',
      description: '',
      name: '')
  }
}

pipeline {
  steps {
    dirs('tmp') {
      script {
        for(e in ary) {
          break ;
        }
      }
    }
  }
}

pipeline {
  steps {
    dirs('tmp') {
      script {
        sed "echo \"hello\""
      }
    }
  }
}

pipeline {
  steps {
    dirs('tmp') {
      script {
        try {
        } catch(Exception e) {
        }
      }
    }
  }
}

pipeline {
  steps {
    dirs('tmp') {
      script {
        ['a', 'b', 'c'].each { x ->
          try {
          } catch(Exception e) {
          }
        }
      }
    }
  }
}

pipeline {
  steps {
    dirs('tmp') {
      script {
        for(i = 0 ; i < 10 ; i ++) {

        }
      }
    }
  }
}

def func(a, b, c) {
  stage('tmp') {
    parallel(
      a: a,
      b: b,
      c: c
    )
  }
}

pipeline {
  agent { label "${params.label}" }
}

pipeline {
  post {
    success {
      script {
        dir('xxx') {
          def commit_id = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
        }
      }

      script {
        withEnv(['a=1', 'b=2']) {
          echo "xxx"
        }
      }
    }

    failure {
    }
  }
}

pipeline {
  script {
    def ary =[
      'a',
    ]
    def dict =[
      a: 'a',
    ]
  }
}

pipeline {
  script {
    if (params.xxx == 'abc') {
      echo 'abc'
    } else if (params.xxx == 'def') {
      echo 'def'
    } else {
      echo 'xyz'
    }
  }
}
