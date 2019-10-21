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
steps{
dirs('tmp') {
script {
try {
} catch(Exception e) {
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
