import React from 'react';
import { NavLink } from 'react-router-dom';
import { Avatar, Button, Skeleton } from "@nextui-org/react";
import { FiHome, FiPlusCircle, FiUser, FiSettings, FiLogOut, FiList } from 'react-icons/fi';
import { useAuth0 } from '@auth0/auth0-react';
import { UserRole, hasRole } from '../models/user';
import { useUser } from '../providers/user-provider';
import logo from '../../public/logo.png';

const Sidebar: React.FC = () => {
  const { logout } = useAuth0();
  const { currentUser, isLoading } = useUser();


  const handleLogout = () => {
    localStorage.removeItem('hasVisitedHomePage');
    logout({ logoutParams: { returnTo: window.location.origin } });
  };

  const mainMenuItems = [
    { name: 'Home', icon: <FiHome />, path: '/home' },
    { name: 'Task Management', icon: <FiList />, path: '/home/tasks' },
    { name: 'Create Task', icon: <FiPlusCircle />, path: '/home/create-task' },
    { name: 'Profile', icon: <FiUser />, path: '/home/profile' },
    { name: 'Admin', icon: <FiSettings />, path: '/home/admin', roles: [UserRole.Admin] },
  ]
    .filter((item) => !item.roles || item.roles?.some((role) => hasRole(currentUser, role)));

  return (
    <aside className="w-64 bg-gray-100 flex flex-col min-h-screen">
      <div className="flex items-center gap-2 p-4">
        <img src={logo} alt="Logo" className="w-9 h-9" />
        <h1 className="w-full text-xs font-bold uppercase opacity-100">CLAUDECOLLABORATORS</h1>
      </div>
      <div className="p-4">
        <div className="flex items-center space-x-4 mb-6">
          {isLoading ? (
            <>
              <Skeleton className="w-10 h-10 rounded-full" />
              <div className="flex flex-col space-y-2">
                <Skeleton className="w-24 h-4" />
                <Skeleton className="w-32 h-3" />
              </div>
            </>
          ) : (
            <>
              <Avatar src={currentUser?.picture} name={currentUser?.name} />
              <div className="flex flex-col">
                <span className="text-sm font-semibold">{currentUser?.name}</span>
                <span className="text-xs text-gray-500">{currentUser?.email}</span>
              </div>
            </>
          )}
        </div>
        <nav className="flex-grow">
          <ul className="space-y-2">
            {mainMenuItems.map((item) => (
              <li key={item.name}>
                <NavLink
                  to={item.path}
                  end={item.path === '/home'}
                  className={({ isActive }) =>
                    `flex items-center p-2 text-base font-normal rounded-lg ${
                      isActive ? 'bg-gray-200 text-gray-900' : 'text-gray-700 hover:bg-gray-200'
                    }`
                  }
                >
                  {item.icon}
                  <span className="ml-3">{item.name}</span>
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
      </div>
      <div className="mt-auto p-4">
        <Button
          color="danger"
          variant="light"
          startContent={<FiLogOut />}
          className="w-full"
          onPress={handleLogout}
        >
          Logout
        </Button>
      </div>
    </aside>
  );
};

export default Sidebar;