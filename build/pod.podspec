Pod::Spec.new do |spec|
  spec.name         = 'Geai'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/ethereumai/go-ethereumai'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS EthereumAI Client'
  spec.source       = { :git => 'https://github.com/ethereumai/go-ethereumai.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Geai.framework'

	spec.prepare_command = <<-CMD
    curl https://geaistore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Geai.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
