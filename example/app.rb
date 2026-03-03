require 'bundler/inline'

gemfile do
  source 'https://rubygems.org'

  gem 'sinatra'
  gem 'rackup'
  gem 'webrick'
end

require 'sinatra/base'

class App < Sinatra::Base
  set :port, ENV.fetch('PORT').to_i
  set :quiet, true

  get '/' do
    'Hello, world!'
  end

  get '/debug' do
    binding.irb
    'debugged'
  end
end

App.run!
