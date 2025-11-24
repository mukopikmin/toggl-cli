require 'net/http'
require 'uri'
require 'json'
require 'date'

# TODO: プロジェクトの部分を扱いやすいようにする

PREVIOUS_MONTH_FLAG = '-p'

def main
  config = load_config

  # Parse command line arguments for date range
  if ARGV.length < 2
    puts "Usage: ruby toggl.rb <start_day> <end_day> [#{PREVIOUS_MONTH_FLAG}]"
    puts "Example: ruby toggl.rb 1 15"
    puts "Example: ruby toggl.rb 1 15 #{PREVIOUS_MONTH_FLAG}"
    exit 1
  end

  start_day = ARGV[0].to_i
  end_day = ARGV[1].to_i
  use_previous_month = ARGV.include?(PREVIOUS_MONTH_FLAG)

  # Set dates to current month or previous month
  current_date = Date.today
  target_date = use_previous_month ? current_date << 1 : current_date

  date_from = Date.new(target_date.year, target_date.month, start_day)
  date_to = Date.new(target_date.year, target_date.month, end_day)
  # days = (date_to.mjd - date_from.mjd + 1).times.map { |i| date_from + i }
  days = (date_from..date_to).to_a

  p date_from
  p date_to

  projects = get_projects(config)
  date_entries = get_time_entries_for_days(config, date_from, date_to)
  
  puts '--- Project list ---'
  projects.each { |p| puts p['name'] }
  puts
  puts '--- Work time table (The order of the rows follows the projects list) ---'
  puts days.map { |d| d.strftime("%F") }.join("\t")
  # puts projects.map { |p| days.map { |d| date_entries[d.to_s][p['id']] || '' }.flatten.join("\t") }
  puts projects.map { |project| days.map { |d| date_entries[d.day]&.[](project['id']) || '' }.flatten.join("\t") }
  # puts projects.map { |project| puts "--- #{project} ---"; days.map { |d| puts d; puts date_entries[d.day]; date_entries[d.day]&.[](project['id']) || '' }.flatten.join("\t") }
end

# Get projects from Toggl API
def get_projects(config)
  uri = URI("https://api.track.toggl.com/api/v9/workspaces/#{config['WORKSPACE']}/projects")
  req = Net::HTTP::Get.new(uri)
  req['Content-Type'] = "application/json"
  req.basic_auth(config['TOKEN'], 'api_token')
  res = Net::HTTP.start(uri.host, uri.port, :use_ssl => uri.scheme == 'https') { |http|
    http.request(req)
  }
  JSON.parse(res.body).filter { |p| p['active'] }
end

# Get time entries for specified days
def get_time_entries_for_days(config, from_day, to_day)
  start_time = "#{from_day}T00:00:00+09:00"
  end_time   = "#{to_day + 1}T00:00:00+09:00"

  uri = URI("https://api.track.toggl.com/api/v9/me/time_entries")
  uri.query = URI.encode_www_form({ start_date: start_time, end_date: end_time, meta: 'true' })
  req = Net::HTTP::Get.new(uri)
  req['Content-Type'] = "application/json"
  req.basic_auth(config['TOKEN'], 'api_token')
  res = Net::HTTP.start(uri.host, uri.port, :use_ssl => uri.scheme == 'https') { |http|
    http.request(req)
  }
  # project_time_entries = JSON.parse(res.body)
  #   .group_by { |t| t['project_id'] }
  #   .map{ |key, value| [key, value.map { |v| v['duration']}.sum / 60 ] }
  #   .to_h

  date_entries=JSON.parse(res.body)
    .group_by { |entry|
      # puts DateTime.parse(entry['start']).to_time.localtime.to_date.day
      DateTime.parse(entry['start']).to_time.localtime.to_date.day
    }.map { |key, values| 
      # grouped = values.group_by { |t| t['project_id'] }
      #   .map{ |key, value| [key, value.map { |v| v['duration']}.sum / 60 ] }
      #   .to_h
      [
        key,
        values.group_by { |t| t['project_id'] }
          .map{ |key, value| [key, value.map { |v| v['duration']}.sum / 60 ] }
          .to_h
      ]
  }.to_h

  p date_entries

  date_entries
end

# Load environment variables from config file
def load_config
  config_file = File.expand_path('~/.toggl_config')
  unless File.exist?(config_file)
    puts "Error: ~/.toggl_config file not found"
    puts "Please create ~/.toggl_config with the following format:"
    puts "WORKSPACE=your_workspace_id"
    puts "TOKEN=your_api_token"
    exit 1
  end
  
  config = {}
  File.readlines(config_file).each do |line|
    line.strip!
    next if line.empty? || line.start_with?('#')
    
    if line.include?('=')
      key, value = line.split('=', 2)
      config[key.strip] = value.strip
    end
  end
  
  # Check if required settings are available
  required_keys = ['WORKSPACE', 'TOKEN']
  missing_keys = required_keys - config.keys
  if missing_keys.any?
    puts "Error: Missing required configuration in ~/.toggl_config: #{missing_keys.join(', ')}"
    exit 1
  end
  
  config
end

main
